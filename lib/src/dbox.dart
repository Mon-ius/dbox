import 'dart:io';
import 'dart:ffi';
import 'dbox_bindings.dart';

import 'package:ffi/ffi.dart';

const String _libName = 'dbox';


class DoxLibrary {
  final dbox _bindings;

  factory DoxLibrary() {
    return DoxLibrary.fromDynamicLibrary(_dylib());
  }

  DoxLibrary.fromDynamicLibrary(DynamicLibrary library) : _bindings = dbox(library);

  static DynamicLibrary _dylib() {
    final libraryPath = switch (Platform.operatingSystem) {
      'ios' => 'lib$_libName.framework/lib$_libName',
      'android' || 'linux' => 'lib$_libName.so',
      'macos' => 'lib$_libName.dylib',
      'windows' => '$_libName.dll',
      _ => throw UnsupportedError('Unknown platform: ${Platform.operatingSystem}'),
    };
    return DynamicLibrary.open(libraryPath);
  }

  int add(int a, int b) {
    return _bindings.Add(a, b);
  }

  int multiply(int a, int b) {
    return _bindings.Multiply(a, b);
  }

  String helloWorld() {
    final Pointer<Char> result = _bindings.HelloWorld();
    final String message = result.cast<Utf8>().toDartString();
    
    _bindings.FreeString(result);
    return message;
  }

  String base64Decode(String encodedString) {
    final Pointer<Char> encodedCString = encodedString.toNativeUtf8().cast<Char>();
    final Pointer<Char> decodedCString = _bindings.Base64Decode(encodedCString);
    
    final String decodedString = decodedCString.cast<Utf8>().toDartString();
    
    malloc.free(encodedCString);
    _bindings.FreeString(decodedCString);
    return decodedString;
  }

  void printDebug(String message) {
    final Pointer<Char> cMessage = message.toNativeUtf8().cast<Char>();
    _bindings.PrintDebug(cMessage);
  }
}

void main() {
  final dboxLib = DoxLibrary();
  
  print('Add(5, 7) = ${dboxLib.add(5, 7)}');
  print('Multiply(5, 7) = ${dboxLib.multiply(5, 7)}');
  print('HelloWorld() = ${dboxLib.helloWorld()}');
  
  const String encoded = 'SGVsbG8gV29ybGQh';
  print('Encoded: $encoded');
  print('Decoded (Go): ${dboxLib.base64Decode(encoded)}');
  dboxLib.printDebug('Debug message from Dart');
}