import 'dart:convert';
import 'dart:io';

import 'package:flutter/services.dart';
import 'package:http/http.dart' as http;
import 'package:package_info_plus/package_info_plus.dart';
import 'package:path/path.dart' as path;
import 'package:path_provider/path_provider.dart';

class MediaHubUpdate {
  final String version;
  final String downloadUrl;
  final String fileName;

  const MediaHubUpdate({required this.version, required this.downloadUrl, required this.fileName});
}

/// Queries the private builder, independently of Plex/Jellyfin credentials.
class MediaUpdateService {
  static const _builderBaseUrl = 'https://plexr.blazz.cl';
  static const _variant = 'plezy-media-hub';
  static const _installerChannel = MethodChannel('com.plezy/apk_installer');

  static Future<MediaHubUpdate?> checkForUpdate({http.Client? client}) async {
    final httpClient = client ?? http.Client();
    try {
      final package = await PackageInfo.fromPlatform();
      final current = '${package.version}+${package.buildNumber}';
      final uri = Uri.parse('$_builderBaseUrl/api/latest?variant=$_variant');
      final response = await httpClient.get(uri).timeout(const Duration(seconds: 10));
      if (response.statusCode != 200) return null;
      final body = jsonDecode(response.body) as Map<String, dynamic>;
      final version = body['version'] as String?;
      final path = body['url'] as String?;
      final fileName = body['name'] as String?;
      if (version == null || path == null || fileName == null || !_isNewer(version, current)) return null;
      return MediaHubUpdate(version: version, downloadUrl: _builderBaseUrl + path, fileName: fileName);
    } finally {
      if (client == null) httpClient.close();
    }
  }

  static bool isNewerForTest(String candidate, String installed) => _isNewer(candidate, installed);

  /// Downloads the APK into the app cache. The Android installer receives it
  /// through Plezy's existing FileProvider, never through a raw file URI.
  static Future<File> downloadApk(
    MediaHubUpdate update, {
    void Function(int receivedBytes, int totalBytes)? onProgress,
    http.Client? client,
  }) async {
    if (!Platform.isAndroid) {
      throw UnsupportedError('La instalación directa solo está disponible en Android.');
    }
    final safeName = path.basename(update.fileName);
    if (!safeName.toLowerCase().endsWith('.apk')) {
      throw const FormatException('El servidor no entregó un APK válido.');
    }
    final httpClient = client ?? http.Client();
    try {
      final response = await httpClient
          .send(http.Request('GET', Uri.parse(update.downloadUrl)))
          .timeout(const Duration(minutes: 5));
      if (response.statusCode != HttpStatus.ok) {
        throw HttpException('No se pudo descargar la actualización (${response.statusCode}).');
      }
      final destination = File(path.join((await getTemporaryDirectory()).path, safeName));
      final sink = destination.openWrite();
      var received = 0;
      final total = response.contentLength ?? 0;
      try {
        await for (final chunk in response.stream) {
          received += chunk.length;
          sink.add(chunk);
          onProgress?.call(received, total);
        }
      } finally {
        await sink.close();
      }
      if (received == 0) throw HttpException('La descarga llegó vacía.');
      return destination;
    } finally {
      if (client == null) httpClient.close();
    }
  }

  /// Opens Android's package installer. If Android blocks unknown sources, it
  /// opens the correct per-app setting and the user can press Install again.
  static Future<ApkInstallResult> installApk(File apk) async {
    if (!Platform.isAndroid) {
      throw UnsupportedError('La instalación directa solo está disponible en Android.');
    }
    final result = await _installerChannel.invokeMapMethod<String, dynamic>('installApk', apk.path);
    return ApkInstallResult(
      started: result?['started'] == true,
      requiresPermission: result?['requiresPermission'] == true,
    );
  }

  static bool _isNewer(String candidate, String installed) {
    final next = _parse(candidate);
    final current = _parse(installed);
    if (next == null || current == null) return false;
    for (var index = 0; index < next.length; index++) {
      if (next[index] != current[index]) return next[index] > current[index];
    }
    return false;
  }

  static List<int>? _parse(String value) {
    final match = RegExp(r'^(\d+)\.(\d+)\.(\d+)(?:-(\d+))?\+(\d+)$').firstMatch(value.trim());
    if (match == null) return null;
    return List<int>.generate(5, (index) => int.parse(match.group(index + 1) ?? '0'));
  }
}

class ApkInstallResult {
  final bool started;
  final bool requiresPermission;

  const ApkInstallResult({required this.started, required this.requiresPermission});
}
