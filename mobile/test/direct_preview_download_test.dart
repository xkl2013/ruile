import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:ruile_mobile/main.dart';

void main() {
  test('streams a direct preview response to a file', () async {
    final server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
    final payload = List<int>.generate(2 * 1024 * 1024, (index) => index % 251);
    final serverDone = server.first.then((request) async {
      request.response.statusCode = HttpStatus.ok;
      request.response.headers.contentLength = payload.length;
      for (var offset = 0; offset < payload.length; offset += 64 * 1024) {
        final end = (offset + 64 * 1024).clamp(0, payload.length);
        request.response.add(payload.sublist(offset, end));
      }
      await request.response.close();
    });
    addTearDown(() async {
      await server.close(force: true);
      await serverDone;
    });

    final root = await Directory.systemTemp.createTemp('ruile-preview-');
    addTearDown(() => root.delete(recursive: true));
    final destination = File('${root.path}/large.pdf');
    final client = HttpClient();
    addTearDown(() => client.close(force: true));

    final result = await streamDirectPreviewToFile(
      httpClient: client,
      directUrl: 'http://127.0.0.1:${server.port}/large.pdf',
      destination: destination,
    );

    expect(result.path, destination.path);
    expect(await result.length(), payload.length);
    expect(await result.openRead(0, 16).expand((chunk) => chunk).toList(),
        payload.take(16));
  });

  test('reports object storage error details without exposing the signature',
      () async {
    final server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
    final serverDone = server.first.then((request) async {
      request.response.statusCode = HttpStatus.forbidden;
      request.response.write(
        '<Error><Code>SignatureDoesNotMatch</Code>'
        '<Message>The request signature is invalid.</Message></Error>',
      );
      await request.response.close();
    });
    addTearDown(() async {
      await server.close(force: true);
      await serverDone;
    });

    final root = await Directory.systemTemp.createTemp('ruile-preview-error-');
    addTearDown(() => root.delete(recursive: true));
    final client = HttpClient();
    addTearDown(() => client.close(force: true));

    await expectLater(
      streamDirectPreviewToFile(
        httpClient: client,
        directUrl:
            'http://127.0.0.1:${server.port}/audio.mp3?x-oss-signature=secret',
        destination: File('${root.path}/audio.mp3'),
      ),
      throwsA(
        isA<HttpException>()
            .having(
              (error) => error.message,
              'message',
              contains('code=SignatureDoesNotMatch'),
            )
            .having(
              (error) => error.message,
              'signature marker',
              contains('signed=true'),
            )
            .having(
              (error) => error.message,
              'secret',
              isNot(contains('secret')),
            ),
      ),
    );
  });
}
