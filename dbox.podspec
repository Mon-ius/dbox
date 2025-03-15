Pod::Spec.new do |s|
  s.name             = 'DBoxFramework'
  s.version          = '0.1.0'
  s.summary          = 'DBox framework for iOS'
  s.description      = 'A cross-platform library compiled for iOS from Go code.'
  s.homepage         = 'https://github.com/Mon-ius/dbox'
  s.license          = { :type => 'GNU General Public License v3.0', :file => 'LICENSE' }
  s.author           = { 'M0nius' => 'm0niusplus@gmail.com' }
  s.source           = { :git => 'https://github.com/Mon-ius/dbox.git', :tag => s.version.to_s }
  
  s.ios.deployment_target = '13.0'
  s.swift_version = '5.0'
  
  s.vendored_frameworks = 'platforms/ios/libdbox.xcframework'
  
  s.pod_target_xcconfig = { 'EXCLUDED_ARCHS[sdk=iphonesimulator*]' => 'arm64' }
  s.user_target_xcconfig = { 'EXCLUDED_ARCHS[sdk=iphonesimulator*]' => 'arm64' }
  s.requires_arc = true
end
