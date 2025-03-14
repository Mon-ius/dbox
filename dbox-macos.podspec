Pod::Spec.new do |s|
  s.name             = 'DBoxFramework-macOS'
  s.version          = '0.1.0'
  s.summary          = 'DBox framework for macOS'
  s.description      = 'A cross-platform library compiled for macOS from Go code.'
  s.homepage         = 'https://github.com/Mon-ius/dbox'
  s.license          = { :type => 'GNU General Public License v3.0', :file => 'LICENSE' }
  s.author           = { 'M0nius' => 'm0niusplus@gmail.com' }
  s.source           = { :git => 'https://github.com/Mon-ius/dbox.git', :tag => s.version.to_s }
  
  s.osx.deployment_target = '10.13'
  s.swift_version = '5.0'
  
  s.vendored_frameworks = 'macos/DBoxFramework.framework'
  
  s.requires_arc = true
end
