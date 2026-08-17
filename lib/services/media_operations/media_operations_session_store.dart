import 'dart:convert';

import '../../models/media_operations/media_operations_models.dart';
import '../../profiles/profile.dart';
import '../base_shared_preferences_service.dart';
import '../credential_vault.dart';

class MediaOperationsSessionStore {
  static const _baseKey = 'media_operations_session';
  String _key(String profileId) => profileScopedPrefsKey(profileId, _baseKey);

  Future<MediaOperationsSession?> load(String profileId) async {
    final prefs = await BaseSharedPreferencesService.sharedCache();
    final raw = readTolerantString(prefs, _key(profileId));
    if (raw == null) return null;
    try {
      final stored = MediaOperationsSession.fromJson(jsonDecode(raw) as Map<String, dynamic>);
      final apiKey = await CredentialVault.reveal(stored.apiKey);
      return apiKey == null ? null : MediaOperationsSession(baseUrl: stored.baseUrl, apiKey: apiKey);
    } catch (_) {
      return null;
    }
  }

  Future<void> save(String profileId, MediaOperationsSession session) async {
    final prefs = await BaseSharedPreferencesService.sharedCache();
    final stored = MediaOperationsSession(baseUrl: session.baseUrl.trim().replaceFirst(RegExp(r'/+$'), ''), apiKey: await CredentialVault.protect(session.apiKey));
    await prefs.setString(_key(profileId), jsonEncode(stored.toJson()));
  }
}
