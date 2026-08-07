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

  Future<List<IdentityCandidate>> listCandidates({String? status}) async {
    final query = status == null || status.isEmpty ? '' : '?status=$status&limit=500';
    final response = await _send('GET', '/api/v1/candidates$query');
    return _items(response).map(IdentityCandidate.fromJson).toList();
  }

  Future<List<IdentityCandidate>> listPendingCandidates() => listCandidates(status: 'pending_review');

  Future<List<ReorganizationPlan>> listPlans() async {
    final response = await _send('GET', '/api/v1/reorganization-plans?limit=500');
    return _items(response).map(ReorganizationPlan.fromJson).toList();
  }

  Future<List<WorkflowJob>> listJobs() async {
    final response = await _send('GET', '/api/v1/workflow-jobs?limit=500');
    return _items(response).map(WorkflowJob.fromJson).toList();
  }

  Future<DiskHealth> getDiskHealth() async {
    final response = await _send('GET', '/api/v1/monitor/disk');
    return DiskHealth.fromJson(jsonDecode(response.body) as Map<String, dynamic>);
  }

  Future<void> approve(int candidateId) async => _send('POST', '/api/v1/candidates/$candidateId/approve');
  Future<void> reject(int candidateId) async => _send('POST', '/api/v1/candidates/$candidateId/reject');

  List<Map<String, dynamic>> _items(http.Response response) {
    final body = jsonDecode(response.body) as Map<String, dynamic>;
    final items = body['items'] as List<dynamic>? ?? const [];
    return items.map((item) => item as Map<String, dynamic>).toList();
  }

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
