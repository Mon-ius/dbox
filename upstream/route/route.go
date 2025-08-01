package route

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"os"
	"os/user"
	"strings"
	"time"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/conntrack"
	"github.com/sagernet/sing-box/common/process"
	"github.com/sagernet/sing-box/common/sniff"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	R "github.com/sagernet/sing-box/route/rule"
	"github.com/sagernet/sing-dns"
	"github.com/sagernet/sing-mux"
	"github.com/sagernet/sing-vmess"
	"github.com/sagernet/sing/common"
	"github.com/sagernet/sing/common/buf"
	"github.com/sagernet/sing/common/bufio"
	"github.com/sagernet/sing/common/bufio/deadline"
	E "github.com/sagernet/sing/common/exceptions"
	F "github.com/sagernet/sing/common/format"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
	"github.com/sagernet/sing/common/uot"
)

// Deprecated: use RouteConnectionEx instead.
func (r *Router) RouteConnection(ctx context.Context, conn net.Conn, metadata adapter.InboundContext) error {
	done := make(chan interface{})
	err := r.routeConnection(ctx, conn, metadata, N.OnceClose(func(it error) {
		close(done)
	}))
	if err != nil {
		return err
	}
	select {
	case <-done:
	case <-r.ctx.Done():
	}
	return nil
}

func (r *Router) RouteConnectionEx(ctx context.Context, conn net.Conn, metadata adapter.InboundContext, onClose N.CloseHandlerFunc) {
	err := r.routeConnection(ctx, conn, metadata, onClose)
	if err != nil {
		N.CloseOnHandshakeFailure(conn, onClose, err)
		if E.IsClosedOrCanceled(err) || R.IsRejected(err) {
			r.logger.DebugContext(ctx, "connection closed: ", err)
		} else {
			r.logger.ErrorContext(ctx, err)
		}
	}
}

func (r *Router) routeConnection(ctx context.Context, conn net.Conn, metadata adapter.InboundContext, onClose N.CloseHandlerFunc) error {
	//nolint:staticcheck
	if metadata.InboundDetour != "" {
		if metadata.LastInbound == metadata.InboundDetour {
			return E.New("routing loop on detour: ", metadata.InboundDetour)
		}
		detour, loaded := r.inbound.Get(metadata.InboundDetour)
		if !loaded {
			return E.New("inbound detour not found: ", metadata.InboundDetour)
		}
		injectable, isInjectable := detour.(adapter.TCPInjectableInbound)
		if !isInjectable {
			return E.New("inbound detour is not TCP injectable: ", metadata.InboundDetour)
		}
		metadata.LastInbound = metadata.Inbound
		metadata.Inbound = metadata.InboundDetour
		metadata.InboundDetour = ""
		injectable.NewConnectionEx(ctx, conn, metadata, onClose)
		return nil
	}
	conntrack.KillerCheck()
	metadata.Network = N.NetworkTCP
	switch metadata.Destination.Fqdn {
	case mux.Destination.Fqdn:
		return E.New("global multiplex is deprecated since sing-box v1.7.0, enable multiplex in Inbound fields instead.")
	case vmess.MuxDestination.Fqdn:
		return E.New("global multiplex (v2ray legacy) not supported since sing-box v1.7.0.")
	case uot.MagicAddress:
		return E.New("global UoT not supported since sing-box v1.7.0.")
	case uot.LegacyMagicAddress:
		return E.New("global UoT (legacy) not supported since sing-box v1.7.0.")
	}
	if deadline.NeedAdditionalReadDeadline(conn) {
		conn = deadline.NewConn(conn)
	}
	selectedRule, _, buffers, _, err := r.matchRule(ctx, &metadata, false, conn, nil)
	if err != nil {
		return err
	}
	var selectedOutbound adapter.Outbound
	if selectedRule != nil {
		switch action := selectedRule.Action().(type) {
		case *R.RuleActionRoute:
			var loaded bool
			selectedOutbound, loaded = r.outbound.Outbound(action.Outbound)
			if !loaded {
				buf.ReleaseMulti(buffers)
				return E.New("outbound not found: ", action.Outbound)
			}
			if !common.Contains(selectedOutbound.Network(), N.NetworkTCP) {
				buf.ReleaseMulti(buffers)
				return E.New("TCP is not supported by outbound: ", selectedOutbound.Tag())
			}
		case *R.RuleActionReject:
			buf.ReleaseMulti(buffers)
			N.CloseOnHandshakeFailure(conn, onClose, action.Error(ctx))
			return nil
		case *R.RuleActionHijackDNS:
			for _, buffer := range buffers {
				conn = bufio.NewCachedConn(conn, buffer)
			}
			N.CloseOnHandshakeFailure(conn, onClose, r.hijackDNSStream(ctx, conn, metadata))
			return nil
		}
	}
	if selectedRule == nil {
		defaultOutbound := r.outbound.Default()
		if !common.Contains(defaultOutbound.Network(), N.NetworkTCP) {
			buf.ReleaseMulti(buffers)
			return E.New("TCP is not supported by default outbound: ", defaultOutbound.Tag())
		}
		selectedOutbound = defaultOutbound
	}

	for _, buffer := range buffers {
		conn = bufio.NewCachedConn(conn, buffer)
	}
	for _, tracker := range r.trackers {
		conn = tracker.RoutedConnection(ctx, conn, metadata, selectedRule, selectedOutbound)
	}
	if outboundHandler, isHandler := selectedOutbound.(adapter.ConnectionHandlerEx); isHandler {
		outboundHandler.NewConnectionEx(ctx, conn, metadata, onClose)
	} else {
		r.connection.NewConnection(ctx, selectedOutbound, conn, metadata, onClose)
	}
	return nil
}

func (r *Router) RoutePacketConnection(ctx context.Context, conn N.PacketConn, metadata adapter.InboundContext) error {
	done := make(chan interface{})
	err := r.routePacketConnection(ctx, conn, metadata, N.OnceClose(func(it error) {
		close(done)
	}))
	if err != nil {
		conn.Close()
		if E.IsClosedOrCanceled(err) || R.IsRejected(err) {
			r.logger.DebugContext(ctx, "connection closed: ", err)
		} else {
			r.logger.ErrorContext(ctx, err)
		}
	}
	select {
	case <-done:
	case <-r.ctx.Done():
	}
	return nil
}

func (r *Router) RoutePacketConnectionEx(ctx context.Context, conn N.PacketConn, metadata adapter.InboundContext, onClose N.CloseHandlerFunc) {
	err := r.routePacketConnection(ctx, conn, metadata, onClose)
	if err != nil {
		N.CloseOnHandshakeFailure(conn, onClose, err)
		if E.IsClosedOrCanceled(err) || R.IsRejected(err) {
			r.logger.DebugContext(ctx, "connection closed: ", err)
		} else {
			r.logger.ErrorContext(ctx, err)
		}
	}
}

func (r *Router) routePacketConnection(ctx context.Context, conn N.PacketConn, metadata adapter.InboundContext, onClose N.CloseHandlerFunc) error {
	//nolint:staticcheck
	if metadata.InboundDetour != "" {
		if metadata.LastInbound == metadata.InboundDetour {
			return E.New("routing loop on detour: ", metadata.InboundDetour)
		}
		detour, loaded := r.inbound.Get(metadata.InboundDetour)
		if !loaded {
			return E.New("inbound detour not found: ", metadata.InboundDetour)
		}
		injectable, isInjectable := detour.(adapter.UDPInjectableInbound)
		if !isInjectable {
			return E.New("inbound detour is not UDP injectable: ", metadata.InboundDetour)
		}
		metadata.LastInbound = metadata.Inbound
		metadata.Inbound = metadata.InboundDetour
		metadata.InboundDetour = ""
		injectable.NewPacketConnectionEx(ctx, conn, metadata, onClose)
		return nil
	}
	conntrack.KillerCheck()

	// TODO: move to UoT
	metadata.Network = N.NetworkUDP

	// Currently we don't have deadline usages for UDP connections
	/*if deadline.NeedAdditionalReadDeadline(conn) {
		conn = deadline.NewPacketConn(bufio.NewNetPacketConn(conn))
	}*/

	selectedRule, _, _, packetBuffers, err := r.matchRule(ctx, &metadata, false, nil, conn)
	if err != nil {
		return err
	}
	var selectedOutbound adapter.Outbound
	var selectReturn bool
	if selectedRule != nil {
		switch action := selectedRule.Action().(type) {
		case *R.RuleActionRoute:
			var loaded bool
			selectedOutbound, loaded = r.outbound.Outbound(action.Outbound)
			if !loaded {
				N.ReleaseMultiPacketBuffer(packetBuffers)
				return E.New("outbound not found: ", action.Outbound)
			}
			if !common.Contains(selectedOutbound.Network(), N.NetworkUDP) {
				N.ReleaseMultiPacketBuffer(packetBuffers)
				return E.New("UDP is not supported by outbound: ", selectedOutbound.Tag())
			}
		case *R.RuleActionReject:
			N.ReleaseMultiPacketBuffer(packetBuffers)
			N.CloseOnHandshakeFailure(conn, onClose, action.Error(ctx))
			return nil
		case *R.RuleActionHijackDNS:
			r.hijackDNSPacket(ctx, conn, packetBuffers, metadata, onClose)
			return nil
		}
	}
	if selectedRule == nil || selectReturn {
		defaultOutbound := r.outbound.Default()
		if !common.Contains(defaultOutbound.Network(), N.NetworkUDP) {
			N.ReleaseMultiPacketBuffer(packetBuffers)
			return E.New("UDP is not supported by outbound: ", defaultOutbound.Tag())
		}
		selectedOutbound = defaultOutbound
	}
	for _, buffer := range packetBuffers {
		conn = bufio.NewCachedPacketConn(conn, buffer.Buffer, buffer.Destination)
		N.PutPacketBuffer(buffer)
	}
	for _, tracker := range r.trackers {
		conn = tracker.RoutedPacketConnection(ctx, conn, metadata, selectedRule, selectedOutbound)
	}
	if metadata.FakeIP {
		conn = bufio.NewNATPacketConn(bufio.NewNetPacketConn(conn), metadata.OriginDestination, metadata.Destination)
	}
	if outboundHandler, isHandler := selectedOutbound.(adapter.PacketConnectionHandlerEx); isHandler {
		outboundHandler.NewPacketConnectionEx(ctx, conn, metadata, onClose)
	} else {
		r.connection.NewPacketConnection(ctx, selectedOutbound, conn, metadata, onClose)
	}
	return nil
}

func (r *Router) PreMatch(metadata adapter.InboundContext) error {
	selectedRule, _, _, _, err := r.matchRule(r.ctx, &metadata, true, nil, nil)
	if err != nil {
		return err
	}
	if selectedRule == nil {
		return nil
	}
	rejectAction, isReject := selectedRule.Action().(*R.RuleActionReject)
	if !isReject {
		return nil
	}
	return rejectAction.Error(context.Background())
}

func (r *Router) matchRule(
	ctx context.Context, metadata *adapter.InboundContext, preMatch bool,
	inputConn net.Conn, inputPacketConn N.PacketConn,
) (
	selectedRule adapter.Rule, selectedRuleIndex int,
	buffers []*buf.Buffer, packetBuffers []*N.PacketBuffer, fatalErr error,
) {
	if r.processSearcher != nil && metadata.ProcessInfo == nil {
		var originDestination netip.AddrPort
		if metadata.OriginDestination.IsValid() {
			originDestination = metadata.OriginDestination.AddrPort()
		} else if metadata.Destination.IsIP() {
			originDestination = metadata.Destination.AddrPort()
		}
		processInfo, fErr := process.FindProcessInfo(r.processSearcher, ctx, metadata.Network, metadata.Source.AddrPort(), originDestination)
		if fErr != nil {
			r.logger.InfoContext(ctx, "failed to search process: ", fErr)
		} else {
			if processInfo.ProcessPath != "" {
				r.logger.InfoContext(ctx, "found process path: ", processInfo.ProcessPath)
			} else if processInfo.PackageName != "" {
				r.logger.InfoContext(ctx, "found package name: ", processInfo.PackageName)
			} else if processInfo.UserId != -1 {
				if /*needUserName &&*/ true {
					osUser, _ := user.LookupId(F.ToString(processInfo.UserId))
					if osUser != nil {
						processInfo.User = osUser.Username
					}
				}
				if processInfo.User != "" {
					r.logger.InfoContext(ctx, "found user: ", processInfo.User)
				} else {
					r.logger.InfoContext(ctx, "found user id: ", processInfo.UserId)
				}
			}
			metadata.ProcessInfo = processInfo
		}
	}
	if r.fakeIPStore != nil && r.fakeIPStore.Contains(metadata.Destination.Addr) {
		domain, loaded := r.fakeIPStore.Lookup(metadata.Destination.Addr)
		if !loaded {
			fatalErr = E.New("missing fakeip record, try to configure experimental.cache_file")
			return
		}
		metadata.OriginDestination = metadata.Destination
		metadata.Destination = M.Socksaddr{
			Fqdn: domain,
			Port: metadata.Destination.Port,
		}
		metadata.FakeIP = true
		r.logger.DebugContext(ctx, "found fakeip domain: ", domain)
	}
	if r.dnsReverseMapping != nil && metadata.Domain == "" {
		domain, loaded := r.dnsReverseMapping.Query(metadata.Destination.Addr)
		if loaded {
			metadata.Domain = domain
			r.logger.DebugContext(ctx, "found reserve mapped domain: ", metadata.Domain)
		}
	}
	if metadata.Destination.IsIPv4() {
		metadata.IPVersion = 4
	} else if metadata.Destination.IsIPv6() {
		metadata.IPVersion = 6
	}

	//nolint:staticcheck
	if metadata.InboundOptions != common.DefaultValue[option.InboundOptions]() {
		if !preMatch && metadata.InboundOptions.SniffEnabled {
			newBuffer, newPackerBuffers, newErr := r.actionSniff(ctx, metadata, &R.RuleActionSniff{
				OverrideDestination: metadata.InboundOptions.SniffOverrideDestination,
				Timeout:             time.Duration(metadata.InboundOptions.SniffTimeout),
			}, inputConn, inputPacketConn, nil)
			if newErr != nil {
				fatalErr = newErr
				return
			}
			if newBuffer != nil {
				buffers = []*buf.Buffer{newBuffer}
			} else if len(newPackerBuffers) > 0 {
				packetBuffers = newPackerBuffers
			}
		}
		if dns.DomainStrategy(metadata.InboundOptions.DomainStrategy) != dns.DomainStrategyAsIS {
			fatalErr = r.actionResolve(ctx, metadata, &R.RuleActionResolve{
				Strategy: dns.DomainStrategy(metadata.InboundOptions.DomainStrategy),
			})
			if fatalErr != nil {
				return
			}
		}
		if metadata.InboundOptions.UDPDisableDomainUnmapping {
			metadata.UDPDisableDomainUnmapping = true
		}
		metadata.InboundOptions = option.InboundOptions{}
	}

match:
	for currentRuleIndex, currentRule := range r.rules {
		metadata.ResetRuleCache()
		if !currentRule.Match(metadata) {
			continue
		}
		if !preMatch {
			ruleDescription := currentRule.String()
			if ruleDescription != "" {
				r.logger.DebugContext(ctx, "match[", currentRuleIndex, "] ", currentRule, " => ", currentRule.Action())
			} else {
				r.logger.DebugContext(ctx, "match[", currentRuleIndex, "] => ", currentRule.Action())
			}
		} else {
			switch currentRule.Action().Type() {
			case C.RuleActionTypeReject:
				ruleDescription := currentRule.String()
				if ruleDescription != "" {
					r.logger.DebugContext(ctx, "pre-match[", currentRuleIndex, "] ", currentRule, " => ", currentRule.Action())
				} else {
					r.logger.DebugContext(ctx, "pre-match[", currentRuleIndex, "] => ", currentRule.Action())
				}
			}
		}
		var routeOptions *R.RuleActionRouteOptions
		switch action := currentRule.Action().(type) {
		case *R.RuleActionRoute:
			routeOptions = &action.RuleActionRouteOptions
		case *R.RuleActionRouteOptions:
			routeOptions = action
		}
		if routeOptions != nil {
			// TODO: add nat
			if (routeOptions.OverrideAddress.IsValid() || routeOptions.OverridePort > 0) && !metadata.RouteOriginalDestination.IsValid() {
				metadata.RouteOriginalDestination = metadata.Destination
			}
			if routeOptions.OverrideAddress.IsValid() {
				metadata.Destination = M.Socksaddr{
					Addr: routeOptions.OverrideAddress.Addr,
					Port: metadata.Destination.Port,
					Fqdn: routeOptions.OverrideAddress.Fqdn,
				}
				metadata.DestinationAddresses = nil
			}
			if routeOptions.OverridePort > 0 {
				metadata.Destination = M.Socksaddr{
					Addr: metadata.Destination.Addr,
					Port: routeOptions.OverridePort,
					Fqdn: metadata.Destination.Fqdn,
				}
			}
			if routeOptions.NetworkStrategy != nil {
				metadata.NetworkStrategy = routeOptions.NetworkStrategy
			}
			if len(routeOptions.NetworkType) > 0 {
				metadata.NetworkType = routeOptions.NetworkType
			}
			if len(routeOptions.FallbackNetworkType) > 0 {
				metadata.FallbackNetworkType = routeOptions.FallbackNetworkType
			}
			if routeOptions.FallbackDelay != 0 {
				metadata.FallbackDelay = routeOptions.FallbackDelay
			}
			if routeOptions.UDPDisableDomainUnmapping {
				metadata.UDPDisableDomainUnmapping = true
			}
			if routeOptions.UDPConnect {
				metadata.UDPConnect = true
			}
			if routeOptions.UDPTimeout > 0 {
				metadata.UDPTimeout = routeOptions.UDPTimeout
			}
		}
		switch action := currentRule.Action().(type) {
		case *R.RuleActionSniff:
			if !preMatch {
				newBuffer, newPacketBuffers, newErr := r.actionSniff(ctx, metadata, action, inputConn, inputPacketConn, buffers)
				if newErr != nil {
					fatalErr = newErr
					return
				}
				if newBuffer != nil {
					buffers = append(buffers, newBuffer)
				} else if len(newPacketBuffers) > 0 {
					packetBuffers = append(packetBuffers, newPacketBuffers...)
				}
			} else {
				selectedRule = currentRule
				selectedRuleIndex = currentRuleIndex
				break match
			}
		case *R.RuleActionResolve:
			fatalErr = r.actionResolve(ctx, metadata, action)
			if fatalErr != nil {
				return
			}
		}
		actionType := currentRule.Action().Type()
		if actionType == C.RuleActionTypeRoute ||
			actionType == C.RuleActionTypeReject ||
			actionType == C.RuleActionTypeHijackDNS ||
			(actionType == C.RuleActionTypeSniff && preMatch) {
			selectedRule = currentRule
			selectedRuleIndex = currentRuleIndex
			break match
		}
	}
	return
}

func (r *Router) actionSniff(
	ctx context.Context, metadata *adapter.InboundContext, action *R.RuleActionSniff,
	inputConn net.Conn, inputPacketConn N.PacketConn, inputBuffers []*buf.Buffer,
) (buffer *buf.Buffer, packetBuffers []*N.PacketBuffer, fatalErr error) {
	if sniff.Skip(metadata) {
		r.logger.DebugContext(ctx, "sniff skipped due to port considered as server-first")
		return
	} else if metadata.Protocol != "" {
		r.logger.DebugContext(ctx, "duplicate sniff skipped")
		return
	}
	if inputConn != nil {
		if len(action.StreamSniffers) == 0 && len(action.PacketSniffers) > 0 {
			return
		} else if metadata.SniffError != nil && !errors.Is(metadata.SniffError, sniff.ErrNeedMoreData) {
			r.logger.DebugContext(ctx, "packet sniff skipped due to previous error: ", metadata.SniffError)
			return
		}
		var streamSniffers []sniff.StreamSniffer
		if len(action.StreamSniffers) > 0 {
			streamSniffers = action.StreamSniffers
		} else {
			streamSniffers = []sniff.StreamSniffer{
				sniff.TLSClientHello,
				sniff.HTTPHost,
				sniff.StreamDomainNameQuery,
				sniff.BitTorrent,
				sniff.SSH,
				sniff.RDP,
			}
		}
		sniffBuffer := buf.NewPacket()
		err := sniff.PeekStream(
			ctx,
			metadata,
			inputConn,
			inputBuffers,
			sniffBuffer,
			action.Timeout,
			streamSniffers...,
		)
		metadata.SniffError = err
		if err == nil {
			//goland:noinspection GoDeprecation
			if action.OverrideDestination && M.IsDomainName(metadata.Domain) {
				metadata.Destination = M.Socksaddr{
					Fqdn: metadata.Domain,
					Port: metadata.Destination.Port,
				}
			}
			if metadata.Domain != "" && metadata.Client != "" {
				r.logger.DebugContext(ctx, "sniffed protocol: ", metadata.Protocol, ", domain: ", metadata.Domain, ", client: ", metadata.Client)
			} else if metadata.Domain != "" {
				r.logger.DebugContext(ctx, "sniffed protocol: ", metadata.Protocol, ", domain: ", metadata.Domain)
			} else {
				r.logger.DebugContext(ctx, "sniffed protocol: ", metadata.Protocol)
			}
		}
		if !sniffBuffer.IsEmpty() {
			buffer = sniffBuffer
		} else {
			sniffBuffer.Release()
		}
	} else if inputPacketConn != nil {
		if len(action.PacketSniffers) == 0 && len(action.StreamSniffers) > 0 {
			return
		} else if metadata.SniffError != nil && !errors.Is(metadata.SniffError, sniff.ErrNeedMoreData) {
			r.logger.DebugContext(ctx, "packet sniff skipped due to previous error: ", metadata.SniffError)
			return
		}
		var packetSniffers []sniff.PacketSniffer
		if len(action.PacketSniffers) > 0 {
			packetSniffers = action.PacketSniffers
		} else {
			packetSniffers = []sniff.PacketSniffer{
				sniff.DomainNameQuery,
				sniff.QUICClientHello,
				sniff.STUNMessage,
				sniff.UTP,
				sniff.UDPTracker,
				sniff.DTLSRecord,
			}
		}
		for {
			var (
				sniffBuffer = buf.NewPacket()
				destination M.Socksaddr
				done        = make(chan struct{})
				err         error
			)
			go func() {
				sniffTimeout := C.ReadPayloadTimeout
				if action.Timeout > 0 {
					sniffTimeout = action.Timeout
				}
				inputPacketConn.SetReadDeadline(time.Now().Add(sniffTimeout))
				destination, err = inputPacketConn.ReadPacket(sniffBuffer)
				inputPacketConn.SetReadDeadline(time.Time{})
				close(done)
			}()
			select {
			case <-done:
			case <-ctx.Done():
				inputPacketConn.Close()
				fatalErr = ctx.Err()
				return
			}
			if err != nil {
				sniffBuffer.Release()
				if !errors.Is(err, os.ErrDeadlineExceeded) {
					fatalErr = err
					return
				}
			} else {
				if len(packetBuffers) > 0 || metadata.SniffError != nil {
					err = sniff.PeekPacket(
						ctx,
						metadata,
						sniffBuffer.Bytes(),
						sniff.QUICClientHello,
					)
				} else {
					err = sniff.PeekPacket(
						ctx, metadata,
						sniffBuffer.Bytes(),
						packetSniffers...,
					)
				}
				packetBuffer := N.NewPacketBuffer()
				*packetBuffer = N.PacketBuffer{
					Buffer:      sniffBuffer,
					Destination: destination,
				}
				packetBuffers = append(packetBuffers, packetBuffer)
				metadata.SniffError = err
				if errors.Is(err, sniff.ErrNeedMoreData) {
					// TODO: replace with generic message when there are more multi-packet protocols
					r.logger.DebugContext(ctx, "attempt to sniff fragmented QUIC client hello")
					continue
				}
				if metadata.Protocol != "" {
					//goland:noinspection GoDeprecation
					if action.OverrideDestination && M.IsDomainName(metadata.Domain) {
						metadata.Destination = M.Socksaddr{
							Fqdn: metadata.Domain,
							Port: metadata.Destination.Port,
						}
					}
					if metadata.Domain != "" && metadata.Client != "" {
						r.logger.DebugContext(ctx, "sniffed packet protocol: ", metadata.Protocol, ", domain: ", metadata.Domain, ", client: ", metadata.Client)
					} else if metadata.Domain != "" {
						r.logger.DebugContext(ctx, "sniffed packet protocol: ", metadata.Protocol, ", domain: ", metadata.Domain)
					} else if metadata.Client != "" {
						r.logger.DebugContext(ctx, "sniffed packet protocol: ", metadata.Protocol, ", client: ", metadata.Client)
					} else {
						r.logger.DebugContext(ctx, "sniffed packet protocol: ", metadata.Protocol)
					}
				}
			}
			break
		}
	}
	return
}

func (r *Router) actionResolve(ctx context.Context, metadata *adapter.InboundContext, action *R.RuleActionResolve) error {
	if metadata.Destination.IsFqdn() {
		metadata.DNSServer = action.Server
		addresses, err := r.Lookup(adapter.WithContext(ctx, metadata), metadata.Destination.Fqdn, action.Strategy)
		if err != nil {
			return err
		}
		metadata.DestinationAddresses = addresses
		r.dnsLogger.DebugContext(ctx, "resolved [", strings.Join(F.MapToString(metadata.DestinationAddresses), " "), "]")
		if metadata.Destination.IsIPv4() {
			metadata.IPVersion = 4
		} else if metadata.Destination.IsIPv6() {
			metadata.IPVersion = 6
		}
	}
	return nil
}
