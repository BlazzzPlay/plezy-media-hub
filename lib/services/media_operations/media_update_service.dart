import 'dart:convert';

import 'package:http/http.dart' as http;
import 'package:package_info_plus/package_info_plus.dart';

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

  static Future<MediaHubUpdate?> checkForUpdate({http.Client? client}) async {
    final httpClient = client ?? http.Client();
    try {
      final package = await PackageInfo.fromPlatform();
      final current = package.version + '+' + package.buildNumber;
      final uri = Uri.parse(_builderBaseUrl + '/api/latest?variant=' + _variant);
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
