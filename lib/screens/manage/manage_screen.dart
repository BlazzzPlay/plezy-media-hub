import 'dart:io';

import 'package:flutter/material.dart';
import 'package:material_symbols_icons/symbols.dart';
import 'package:provider/provider.dart';

import '../../models/media_operations/media_operations_models.dart';
import '../../profiles/active_profile_provider.dart';
import '../../services/media_operations/media_operations_http_client.dart';
import '../../services/media_operations/media_operations_session_store.dart';
import '../../services/media_operations/media_update_service.dart';
import '../../widgets/app_icon.dart';

class ManageScreen extends StatefulWidget {
  const ManageScreen({super.key});
  @override
  State<ManageScreen> createState() => _ManageScreenState();
}

class _ManageScreenState extends State<ManageScreen> {
  final _store = MediaOperationsSessionStore();
  final _url = TextEditingController();
  final _key = TextEditingController();
  static const _defaultOrchestratorUrl = 'http://100.87.101.20:8100';
  bool _loading = true;
  bool _saving = false;
  bool _checkingUpdate = false;
  bool _downloadingUpdate = false;
  double? _updateProgress;
  File? _downloadedApk;
  bool _connectionExpanded = false;
  bool _hasSavedConnection = false;
  String? _error;
  MediaHubUpdate? _update;
  List<IdentityCandidate> _candidates = const [];
  List<IdentityCandidate> _historyCandidates = const [];
  List<ReorganizationPlan> _plans = const [];
  List<WorkflowJob> _jobs = const [];
  DiskHealth? _diskHealth;
  bool _orchestratorConnected = false;

  String get _profileId => context.read<ActiveProfileProvider>().active?.id ?? 'local-admin';

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _load();
      _checkForUpdate();
    });
  }

  @override
  void dispose() {
    _url.dispose();
    _key.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    final session = await _store.load(_profileId);
    if (!mounted) return;
    if (session == null) {
      _url.text = _defaultOrchestratorUrl;
      setState(() => _loading = false);
      return;
    }
    _url.text = session.baseUrl;
    _key.text = session.apiKey;
    setState(() => _hasSavedConnection = true);
    await _refresh(session: session);
  }

  Future<void> _refresh({MediaOperationsSession? session}) async {
    final configured = session ?? MediaOperationsSession(baseUrl: _url.text.trim(), apiKey: _key.text.trim());
    if (configured.baseUrl.isEmpty || configured.apiKey.isEmpty) {
      setState(() {
        _loading = false;
        _error = 'Configurá la URL y la clave del Orquestador.';
      });
      return;
    }
    setState(() {
      _loading = true;
      _error = null;
    });
    final client = MediaOperationsHttpClient(configured);
    try {
      await client.checkHealth();
      final results = await Future.wait([
        client.listPendingCandidates(),
        client.listCandidates(),
        client.listPlans(),
        client.listJobs(),
        client.getDiskHealth(),
      ]);
      if (!mounted) {
        return;
      }
      final allCandidates = results[1] as List<IdentityCandidate>;
      setState(() {
        _candidates = results[0] as List<IdentityCandidate>;
        _historyCandidates = allCandidates
            .where((candidate) => candidate.status != 'pending_review' && candidate.status != 'pending')
            .toList();
        _plans = results[2] as List<ReorganizationPlan>;
        _jobs = results[3] as List<WorkflowJob>;
        _diskHealth = results[4] as DiskHealth;
        _orchestratorConnected = true;
      });
    } catch (error) {
      if (mounted) {
        setState(() {
          _error = error.toString();
          _orchestratorConnected = false;
        });
      }
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
      if (mounted) {
        setState(() {
          _hasSavedConnection = true;
          _connectionExpanded = false;
        });
      }
      await _refresh(session: session);
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  Future<void> _checkForUpdate() async {
    if (mounted) setState(() => _checkingUpdate = true);
    try {
      final update = await MediaUpdateService.checkForUpdate();
      if (mounted) setState(() => _update = update);
    } finally {
      if (mounted) setState(() => _checkingUpdate = false);
    }
  }

  Future<void> _downloadAndInstallUpdate() async {
    final update = _update;
    if (update == null) return;
    setState(() {
      _downloadingUpdate = true;
      _updateProgress = 0;
    });
    try {
      final apk =
          _downloadedApk ??
          await MediaUpdateService.downloadApk(
            update,
            onProgress: (received, total) {
              if (mounted) setState(() => _updateProgress = total > 0 ? received / total : null);
            },
          );
      if (!mounted) return;
      setState(() => _downloadedApk = apk);
      final install = await MediaUpdateService.installApk(apk);
      if (!mounted) return;
      if (install.requiresPermission) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(const SnackBar(content: Text('Permití instalar desde Plezy y luego tocá Instalar nuevamente.')));
      }
    } catch (error) {
      if (mounted) ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('No se pudo actualizar: $error')));
    } finally {
      if (mounted) setState(() => _downloadingUpdate = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final groups = <String, List<IdentityCandidate>>{};
    for (final candidate in _candidates) {
      groups.putIfAbsent(_contentGroupKey(candidate), () => []).add(candidate);
    }
    return DefaultTabController(
      length: 6,
      child: Scaffold(
        appBar: AppBar(
          title: const Text('Gestionar biblioteca'),
          actions: [
            IconButton(
              tooltip: 'Configuración del Orquestador',
              onPressed: () => setState(() => _connectionExpanded = !_connectionExpanded),
              icon: const AppIcon(Symbols.settings_rounded),
            ),
            IconButton(onPressed: _loading ? null : _refresh, icon: const AppIcon(Symbols.refresh_rounded)),
          ],
        ),
        body: Column(
          children: [
            if (!_hasSavedConnection || _connectionExpanded) _connectionCard(),
            _updateCard(),
            const TabBar(
              isScrollable: true,
              tabs: [
                Tab(text: 'Resumen'),
                Tab(text: 'Identificar'),
                Tab(text: 'Planes'),
                Tab(text: 'Cola'),
                Tab(text: 'Historial'),
                Tab(text: 'Salud'),
              ],
            ),
            Expanded(
              child: _loading
                  ? const Center(child: CircularProgressIndicator())
                  : _error != null
                  ? _messageCard(icon: Symbols.cloud_off_rounded, title: 'No se pudo cargar', body: _error!)
                  : TabBarView(
                      children: [
                        _summaryView(groups),
                        _reviewView(groups),
                        _plansView(),
                        _queueView(),
                        _historyView(),
                        _healthView(),
                      ],
                    ),
            ),
          ],
        ),
      ),
    );
  }

  String _contentGroupKey(IdentityCandidate candidate) {
    final fileName = candidate.path.split(RegExp(r'[\\/]')).last.replaceFirst(RegExp(r'\.[^.]+$'), '');
    final seriesName = fileName.replaceFirst(RegExp(r'\bS\d{1,2}E\d{1,3}\b.*$', caseSensitive: false), '').replaceAll(RegExp(r'[.\-_\s]+$'), '').trim();
    return seriesName.isEmpty ? 'file:${candidate.inventoryFileId}' : 'series:${seriesName.toLowerCase()}';
  }

  Widget _summaryView(Map<String, List<IdentityCandidate>> groups) {
    final activeJobs = _jobs.where((job) => job.status == 'pending' || job.status == 'running').length;
    final planned = _plans.where((plan) => plan.action == 'planned').length;
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Text('Resumen operativo', style: Theme.of(context).textTheme.titleLarge),
        const SizedBox(height: 12),
        _metricCard(Symbols.manage_search_rounded, 'Identidades', '${groups.length} contenidos pendientes'),
        _metricCard(Symbols.rule_rounded, 'Planes', '$planned por revisar'),
        _metricCard(Symbols.sync_rounded, 'Procesos', '$activeJobs en cola o ejecución'),
        _metricCard(
          _orchestratorConnected ? Symbols.check_circle_rounded : Symbols.error_rounded,
          'Orquestador',
          _orchestratorConnected ? 'NUC conectado y base disponible' : 'Sin conexión',
        ),
      ],
    );
  }

  Widget _healthView() {
    final disk = _diskHealth;
    if (!_orchestratorConnected || disk == null) {
      return _messageCard(
        icon: Symbols.cloud_off_rounded,
        title: 'NUC sin conexión',
        body: 'Guardá la conexión y verificá Tailscale para consultar la salud del Orquestador.',
      );
    }
    final usedPercent = disk.totalGb <= 0 ? 0.0 : disk.usedGb / disk.totalGb;
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Text('Salud del NUC', style: Theme.of(context).textTheme.titleLarge),
        const SizedBox(height: 12),
        Card(
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const ListTile(
                  contentPadding: EdgeInsets.zero,
                  leading: AppIcon(Symbols.check_circle_rounded),
                  title: Text('API y PostgreSQL operativos'),
                  subtitle: Text('Conexión validada desde la app'),
                ),
                Text('Espacio de trabajo: ${disk.path}'),
                const SizedBox(height: 8),
                LinearProgressIndicator(value: usedPercent.clamp(0.0, 1.0)),
                const SizedBox(height: 8),
                Text('${disk.availableGb.toStringAsFixed(1)} GB libres de ${disk.totalGb.toStringAsFixed(1)} GB'),
              ],
            ),
          ),
        ),
      ],
    );
  }

  Widget _metricCard(IconData icon, String title, String value) => Card(
    child: ListTile(leading: AppIcon(icon), title: Text(title), subtitle: Text(value)),
  );

  Widget _reviewView(Map<String, List<IdentityCandidate>> groups) {
    if (groups.isEmpty) {
      return _messageCard(
        icon: Symbols.task_alt_rounded,
        title: 'Sin identidades pendientes',
        body: 'Las aprobaciones quedan en Historial. Antes de mover archivos se debe revisar un plan.',
      );
    }
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Text('${groups.length} contenidos para revisar', style: Theme.of(context).textTheme.titleLarge),
        const SizedBox(height: 8),
        for (final group in groups.values) _candidateGroup(group),
      ],
    );
  }

  Widget _connectionCard() => Card(
    child: Padding(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text('Media Orchestrator', style: Theme.of(context).textTheme.titleMedium),
          const SizedBox(height: 8),
          TextField(
            controller: _url,
            keyboardType: TextInputType.url,
            decoration: const InputDecoration(labelText: 'URL del NUC/NAS', hintText: 'http://100.x.x.x:8100'),
          ),
          const SizedBox(height: 8),
          TextField(
            controller: _key,
            obscureText: true,
            decoration: const InputDecoration(labelText: 'Clave de acceso'),
          ),
          const SizedBox(height: 12),
          FilledButton.icon(
            onPressed: _saving ? null : _saveAndConnect,
            icon: const AppIcon(Symbols.link_rounded),
            label: Text(_saving ? 'Guardando…' : 'Guardar y conectar'),
          ),
        ],
      ),
    ),
  );

  Widget _updateCard() => Card(
    margin: const EdgeInsets.fromLTRB(12, 4, 12, 2),
    child: _update == null
        ? SizedBox(
            height: 48,
            child: ListTile(
              dense: true,
              leading: const AppIcon(Symbols.system_update_rounded),
              title: const Text('Actualizaciones'),
              trailing: IconButton(
                tooltip: 'Buscar actualización',
                onPressed: _checkingUpdate ? null : _checkForUpdate,
                icon: _checkingUpdate
                    ? const SizedBox(width: 18, height: 18, child: CircularProgressIndicator(strokeWidth: 2))
                    : const AppIcon(Symbols.refresh_rounded),
              ),
            ),
          )
        : Padding(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
            child: Column(
              children: [
                Row(
                  children: [
                    const AppIcon(Symbols.system_update_rounded),
                    const SizedBox(width: 12),
                    Expanded(
                      child: Text('Nueva versión: ${_update!.version}', maxLines: 1, overflow: TextOverflow.ellipsis),
                    ),
                    IconButton(
                      tooltip: 'Buscar actualización',
                      onPressed: _checkingUpdate ? null : _checkForUpdate,
                      icon: const AppIcon(Symbols.refresh_rounded),
                    ),
                  ],
                ),
                if (_downloadingUpdate) ...[
                  const SizedBox(height: 6),
                  LinearProgressIndicator(value: _updateProgress),
                  const SizedBox(height: 6),
                ],
                SizedBox(
                  width: double.infinity,
                  child: FilledButton.icon(
                    onPressed: _downloadingUpdate ? null : _downloadAndInstallUpdate,
                    icon: _downloadingUpdate
                        ? const SizedBox(width: 18, height: 18, child: CircularProgressIndicator(strokeWidth: 2))
                        : AppIcon(_downloadedApk == null ? Symbols.download_rounded : Symbols.install_mobile_rounded),
                    label: Text(
                      _downloadingUpdate
                          ? 'Descargando…'
                          : _downloadedApk == null
                          ? 'Descargar e instalar'
                          : 'Instalar actualización',
                    ),
                  ),
                ),
              ],
            ),
          ),
  );

  Widget _plansView() {
    final plans = _plans.where((plan) => plan.action == 'planned').toList();
    if (plans.isEmpty) {
      return _messageCard(
        icon: Symbols.rule_rounded,
        title: 'Sin planes pendientes',
        body: 'Los planes se revisan aquí antes de ejecutar cualquier movimiento.',
      );
    }
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Text('${plans.length} planes para revisar', style: Theme.of(context).textTheme.titleLarge),
        for (final plan in plans)
          Card(
            child: ListTile(
              leading: const AppIcon(Symbols.drive_file_move_rounded),
              title: Text(plan.sourcePath),
              subtitle: Text(
                '${plan.operation.toUpperCase()} → ${plan.targetPath}\n${plan.reason}',
                maxLines: 3,
                overflow: TextOverflow.ellipsis,
              ),
            ),
          ),
      ],
    );
  }

  Widget _queueView() {
    final active = _jobs.where((job) => job.status == 'pending' || job.status == 'running').toList();
    if (active.isEmpty) {
      return _messageCard(
        icon: Symbols.hourglass_empty_rounded,
        title: 'Cola vacía',
        body: 'No hay procesos ejecutándose. Los planes requieren confirmación explícita.',
      );
    }
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Text('${active.length} procesos activos', style: Theme.of(context).textTheme.titleLarge),
        for (final job in active)
          Card(
            child: ListTile(
              leading: AppIcon(job.status == 'running' ? Symbols.sync_rounded : Symbols.schedule_rounded),
              title: Text(job.fileName.isEmpty ? job.type : job.fileName, maxLines: 2, overflow: TextOverflow.ellipsis),
              subtitle: Text('${job.type} · ${job.status} · intento ${job.attempts}/${job.maxAttempts}'),
            ),
          ),
      ],
    );
  }

  Widget _historyView() {
    final plans = _plans.where((plan) => plan.action != 'planned').toList();
    final jobs = _jobs.where((job) => job.status != 'pending' && job.status != 'running').toList();
    if (_historyCandidates.isEmpty && plans.isEmpty && jobs.isEmpty) {
      return _messageCard(
        icon: Symbols.history_rounded,
        title: 'Sin historial',
        body: 'Acá quedarán identidades, planes y procesos finalizados.',
      );
    }
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        for (final candidate in _historyCandidates)
          Card(
            child: ListTile(
              leading: AppIcon(candidate.status == 'accepted' ? Symbols.check_circle_rounded : Symbols.cancel_rounded),
              title: Text(candidate.title),
              subtitle: Text(candidate.status),
            ),
          ),
        for (final plan in plans)
          Card(
            child: ListTile(
              leading: const AppIcon(Symbols.rule_rounded),
              title: Text(plan.action),
              subtitle: Text(plan.sourcePath),
            ),
          ),
        for (final job in jobs)
          Card(
            child: ListTile(
              leading: AppIcon(job.status == 'completed' ? Symbols.check_circle_rounded : Symbols.error_rounded),
              title: Text(job.type),
              subtitle: Text(job.lastError.isEmpty ? job.status : '${job.status} · ${job.lastError}'),
            ),
          ),
      ],
    );
  }

  Future<void> _reviewGroup(List<IdentityCandidate> group, IdentityCandidate selected, bool approved) async {
    final selection = group.where((candidate) => selected.externalId.isNotEmpty
        ? candidate.provider == selected.provider && candidate.externalId == selected.externalId
        : candidate.provider == selected.provider && candidate.title == selected.title && candidate.year == selected.year).toList();
    if (selection.isEmpty) return;
    final client = MediaOperationsHttpClient(MediaOperationsSession(baseUrl: _url.text.trim(), apiKey: _key.text.trim()));
    try {
      await Future.wait(selection.map((candidate) => approved ? client.approve(candidate.id) : client.reject(candidate.id)));
      if (mounted) await _refresh();
    } catch (error) {
      if (mounted) ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(error.toString())));
    } finally {
      client.dispose();
    }
  }

  Widget _candidateGroup(List<IdentityCandidate> items) {
    final first = items.first;
    final fileCount = items.map((candidate) => candidate.inventoryFileId).toSet().length;
    final externalMatches = items.where((candidate) => candidate.externalId.isNotEmpty).toList();
    final choices = (externalMatches.isEmpty ? items : externalMatches).fold<Map<String, IdentityCandidate>>({}, (result, candidate) {
      final key = '${candidate.provider}|${candidate.externalId}|${candidate.title}|${candidate.year}';
      result.putIfAbsent(key, () => candidate);
      return result;
    }).values.toList();
    final title = first.path.split(RegExp(r'[\\/]')).last.replaceFirst(RegExp(r'\.[^.]+$'), '').replaceFirst(RegExp(r'\bS\d{1,2}E\d{1,3}\b.*$', caseSensitive: false), '').replaceAll(RegExp(r'[.\-_\s]+$'), '').trim();
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              title.isEmpty ? first.path.split(RegExp(r'[\\/]')).last : title,
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 8),
            Text('$fileCount archivo${fileCount == 1 ? '' : 's'} · ${choices.length} identidad${choices.length == 1 ? '' : 'es'} posibles'),
            const Divider(),
            for (final candidate in choices)
              ListTile(
                contentPadding: EdgeInsets.zero,
                title: Text(candidate.title),
                subtitle: Text(
                  [
                    candidate.year?.toString(),
                    candidate.provider.toUpperCase(),
                    if (candidate.edition.isNotEmpty) candidate.edition,
                  ].whereType<String>().join(' · '),
                ),
                trailing: Wrap(
                  spacing: 4,
                  children: [
                    IconButton(
                      tooltip: 'Rechazar',
                      onPressed: () => _reviewGroup(items, candidate, false),
                      icon: const AppIcon(Symbols.close_rounded),
                    ),
                    IconButton(
                      tooltip: 'Aprobar',
                      onPressed: () => _reviewGroup(items, candidate, true),
                      icon: const AppIcon(Symbols.check_rounded),
                    ),
                  ],
                ),
              ),
          ],
        ),
      ),
    );
  }

  Widget _messageCard({required IconData icon, required String title, required String body}) => Card(
    child: Padding(
      padding: const EdgeInsets.all(32),
      child: Column(
        children: [
          AppIcon(icon, size: 42),
          const SizedBox(height: 12),
          Text(title, style: Theme.of(context).textTheme.titleLarge),
          const SizedBox(height: 8),
          Text(body, textAlign: TextAlign.center),
        ],
      ),
    ),
  );
}
