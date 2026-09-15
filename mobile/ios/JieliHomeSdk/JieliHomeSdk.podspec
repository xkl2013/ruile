Pod::Spec.new do |s|
  s.name = 'JieliHomeSdk'
  s.version = '1.14.0'
  s.summary = 'Local Jieli Home iOS SDK frameworks.'
  s.description = 'Vendored Jieli Home SDK frameworks used by the Flutter memory card bridge.'
  s.homepage = 'https://doc.zh-jieli.com/Apps/iOS/jielihome/zh-cn/master/index.html'
  s.license = { :type => 'Commercial', :text => 'Provided by the Jieli vendor SDK bundle.' }
  s.author = { 'Jieli' => 'https://www.zh-jieli.com' }
  s.source = { :path => '.' }
  s.platform = :ios, '15.0'
  s.swift_version = '5.0'
  s.vendored_frameworks = 'Frameworks/*.xcframework'
  s.frameworks = 'AVFoundation', 'CoreBluetooth', 'CoreFoundation', 'CoreGraphics', 'UIKit'
  s.libraries = 'c++', 'z', 'bz2'
end
