import 'package:flutter/material.dart';
import 'package:material_symbols_icons/symbols.dart';
import 'package:provider/provider.dart';

import '../../models/media_operations/media_operations_models.dart';
import '../../profiles/active_profile_provider.dart';
import '../../services/media_operations/media_operations_http_client.dart';
import '../../services/media_operations/media_operations_session_store.dart';
import '../../widgets/app_icon.dart';

class ManageScreen extends StatefulWidget {
  const ManageScreen({super.key});
  @override State<ManageScreen> createState() => _ManageScreenState();
}

class _ManageScreenState extends State<ManageScreen> {
  final _store = MediaOperationsSessionStore();
  final _url = TextEditingController();
  final _key = TextEditingController();
  static const _defaultOrchestratorUrl = 'http://100.87.101.20:8100';
  bool _loading = true;
  bool _saving = false;
  String? _error;
  List<IdentityCandidate> _candidates = const [];

  String get _profileId => context.read<ActiveProfileProvider>().active?.id ?? 'local-admin';

  @override void initState() { super.initState(); WidgetsBinding.instance.addPostFrameCallback((_) => _load()); }
  @override void dispose() { _url.dispose(); _key.dispose(); super.dispose(); }

  Future<void> _load() async {
    setState(() { _loading = true; _error = null; });
    final session = await _store.load(_profileId);
    if (!mounted) return;
    if (session == null) {
      _url.text = _defaultOrchestratorUrl;
      setState(() => _loading = false);
      return;
    }
    _url.text = session.baseUrl;
    _key.text = session.apiKey;
    await _refresh(session: session);
  }

  Future<void> _refresh({MediaOperationsSession? session}) async {
    final configured = session ?? MediaOperationsSession(baseUrl: _url.text.trim(), apiKey: _key.text.trim());
    if (configured.baseUrl.isEmpty || configured.apiKey.isEmpty) { setState(() { _loading = false; _error = 'Configurá la URL y la clave del Orquestador.'; }); return; }
    setState(() { _loading = true; _error = null; });
    final client = MediaOperationsHttpClient(configured);
    try {
      await client.checkHealth();
      final candidates = await client.listPendingCandidates();
      if (!mounted) return;
      setState(() => _candidates = candidates);
    } catch (error) {
      if (mounted) setState(() => _error = error.toString());
    } finally {
      client.dispose();
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _saveAndConnect() async {
    final session = MediaOperationsSession(baseUrl: _url.text.trim(), apiKey: _key.text.trim());
    setState(() => _saving = true);
    try {
      await _store.save(_profileId, session);
      await _refresh(session: session);
    } finally { if (mounted) setState(() => _saving = false); }
  }

  Future<void> _review(IdentityCandidate candidate, bool approved) async {
    final client = MediaOperationsHttpClient(MediaOperationsSession(baseUrl: _url.text.trim(), apiKey: _key.text.trim()));
    try {
      if (approved) { await client.approve(candidate.id); } else { await client.reject(candidate.id); }
      if (mounted) await _refresh();
    } catch (error) {
      if (mounted) ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(error.toString())));
    } finally { client.dispose(); }
  }

  @override Widget build(BuildContext context) {
    final groups = <int, List<IdentityCandidate>>{};
    for (final candidate in _candidates) { groups.putIfAbsent(candidate.inventoryFileId, () => []).add(candidate); }
    return Scaffold(
      appBar: AppBar(title: const Text('Gestionar biblioteca'), actions: [IconButton(onPressed: _loading ? null : _refresh, icon: const AppIcon(Symbols.refresh_rounded))]),
      body: ListView(padding: const EdgeInsets.all(16), children: [
        _connectionCard(), const SizedBox(height: 16),
        if (_loading) const Center(child: Padding(padding: EdgeInsets.all(32), child: CircularProgressIndicator()))
        else if (_error != null) _messageCard(icon: Symbols.cloud_off_rounded, title: 'No se pudo cargar', body: _error!)
        else if (_candidates.isEmpty) _messageCard(icon: Symbols.task_alt_rounded, title: 'Sin identidades pendientes', body: 'El Orquestador está conectado y no hay archivos esperando aprobación.')
        else ...[Text('${groups.length} archivos para revisar', style: Theme.of(context).textTheme.titleLarge), const SizedBox(height: 8), for (final group in groups.values) _candidateGroup(group)],
      ]),
    );
  }

  Widget _connectionCard() => Card(child: Padding(padding: const EdgeInsets.all(16), child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
    Text('Media Orchestrator', style: Theme.of(context).textTheme.titleMedium), const SizedBox(height: 8),
    TextField(controller: _url, keyboardType: TextInputType.url, decoration: const InputDecoration(labelText: 'URL del NUC/NAS', hintText: 'http://100.x.x.x:8100')),
    const SizedBox(height: 8), TextField(controller: _key, obscureText: true, decoration: const InputDecoration(labelText: 'Clave de acceso')),
    const SizedBox(height: 12), FilledButton.icon(onPressed: _saving ? null : _saveAndConnect, icon: const AppIcon(Symbols.link_rounded), label: Text(_saving ? 'Guardando…' : 'Guardar y conectar')),
  ])));

  Widget _candidateGroup(List<IdentityCandidate> items) {
    final first = items.first;
    return Card(child: Padding(padding: const EdgeInsets.all(16), child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      Text(first.path.split(RegExp(r'[\\/]')).last, maxLines: 2, overflow: TextOverflow.ellipsis, style: Theme.of(context).textTheme.titleMedium),
      const SizedBox(height: 8), Text('${items.length} coincidencia${items.length == 1 ? '' : 's'} · elegí una identidad'),
      const Divider(),
      for (final candidate in items) ListTile(contentPadding: EdgeInsets.zero, title: Text(candidate.title), subtitle: Text([candidate.year?.toString(), candidate.provider.toUpperCase(), if (candidate.edition.isNotEmpty) candidate.edition].whereType<String>().join(' · ')), trailing: Wrap(spacing: 4, children: [IconButton(tooltip: 'Rechazar', onPressed: () => _review(candidate, false), icon: const AppIcon(Symbols.close_rounded)), IconButton(tooltip: 'Aprobar', onPressed: () => _review(candidate, true), icon: const AppIcon(Symbols.check_rounded))])),
    ])));
  }

  Widget _messageCard({required IconData icon, required String title, required String body}) => Card(child: Padding(padding: const EdgeInsets.all(32), child: Column(children: [AppIcon(icon, size: 42), const SizedBox(height: 12), Text(title, style: Theme.of(context).textTheme.titleLarge), const SizedBox(height: 8), Text(body, textAlign: TextAlign.center)])));
}
