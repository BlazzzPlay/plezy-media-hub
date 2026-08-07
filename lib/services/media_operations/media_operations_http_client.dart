import 'dart:convert';

import 'package:http/http.dart' as http;

import '../../models/media_operations/media_operations_models.dart';
import '../../utils/abortable_http_request.dart';
import '../../utils/platform_http_client_stub.dart' if (dart.library.io) '../../utils/platform_http_client_io.dart' as platform;

class MediaOperationsHttpClient {
  final MediaOperationsSession session;
  final http.Client _http;

  MediaOperationsHttpClient(this.session, {http.Client? httpClient}) : _http = httpClient ?? platform.createPlatformClient();
  void dispose() => _http.close();

  Future<void> checkHealth() async {
    final response = await sendAbortableHttpRequest(_http, 'GET', Uri.parse('${session.baseUrl}/health'), timeout: const Duration(seconds: 10), operation: 'Media Orchestrator health');
    _throwForStatus(response);
  }

  Future<List<IdentityCandidate>> listPendingCandidates() async {
    final response = await _send('GET', '/api/v1/candidates?status=pending&limit=500');
    final body = jsonDecode(response.body) as Map<String, dynamic>;
    final items = body['items'] as List<dynamic>? ?? const [];
    return items.map((item) => IdentityCandidate.fromJson(item as Map<String, dynamic>)).toList();
  }

  Future<void> approve(int candidateId) async => _send('POST', '/api/v1/candidates/$candidateId/approve');
  Future<void> reject(int candidateId) async => _send('POST', '/api/v1/candidates/$candidateId/reject');

  Future<http.Response> _send(String method, String path) async {
    final response = await sendAbortableHttpRequest(_http, method, Uri.parse('${session.baseUrl}$path'), headers: {'Accept': 'application/json', 'Authorization': 'Bearer ${session.apiKey}'}, timeout: const Duration(seconds: 15), operation: 'Media Orchestrator $method $path');
    _throwForStatus(response);
    return response;
  }

  static void _throwForStatus(http.Response response) {
    if (response.statusCode >= 200 && response.statusCode < 300) return;
    var message = 'HTTP ${response.statusCode}';
    try { message = (jsonDecode(response.body) as Map<String, dynamic>)['error'] as String? ?? message; } catch (_) {}
    throw MediaOperationsException(message);
  }
}

class MediaOperationsException implements Exception {
  final String message;
  const MediaOperationsException(this.message);
  @override String toString() => message;
}
