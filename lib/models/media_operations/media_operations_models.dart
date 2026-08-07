class MediaOperationsSession {
  final String baseUrl;
  final String apiKey;

  const MediaOperationsSession({required this.baseUrl, required this.apiKey});

  Map<String, Object?> toJson() => {'baseUrl': baseUrl, 'apiKey': apiKey};
  factory MediaOperationsSession.fromJson(Map<String, dynamic> json) => MediaOperationsSession(
    baseUrl: json['baseUrl'] as String? ?? '',
    apiKey: json['apiKey'] as String? ?? '',
  );
}

class IdentityCandidate {
  final int id;
  final int inventoryFileId;
  final String path;
  final String provider;
  final String externalId;
  final String title;
  final int? year;
  final String status;
  final String edition;

  const IdentityCandidate({required this.id, required this.inventoryFileId, required this.path, required this.provider, required this.externalId, required this.title, required this.year, required this.status, required this.edition});

  factory IdentityCandidate.fromJson(Map<String, dynamic> json) => IdentityCandidate(
    id: (json['id'] as num?)?.toInt() ?? 0,
    inventoryFileId: (json['inventory_file_id'] as num?)?.toInt() ?? 0,
    path: json['path'] as String? ?? '',
    provider: json['provider'] as String? ?? '',
    externalId: json['external_id'] as String? ?? '',
    title: json['title'] as String? ?? 'Sin título',
    year: (json['year'] as num?)?.toInt(),
    status: json['status'] as String? ?? 'pending',
    edition: json['edition'] as String? ?? '',
  );
}


class ReorganizationPlan {
  final int id;
  final String sourcePath;
  final String targetPath;
  final String operation;
  final String action;
  final String reason;

  const ReorganizationPlan({required this.id, required this.sourcePath, required this.targetPath, required this.operation, required this.action, required this.reason});

  factory ReorganizationPlan.fromJson(Map<String, dynamic> json) => ReorganizationPlan(
    id: (json['id'] as num?)?.toInt() ?? 0,
    sourcePath: json['source_path'] as String? ?? '',
    targetPath: json['target_path'] as String? ?? '',
    operation: json['operation'] as String? ?? 'move',
    action: json['action'] as String? ?? 'planned',
    reason: json['reason'] as String? ?? '',
  );
}

class WorkflowJob {
  final int id;
  final String type;
  final String status;
  final int attempts;
  final int maxAttempts;
  final String lastError;

  const WorkflowJob({required this.id, required this.type, required this.status, required this.attempts, required this.maxAttempts, required this.lastError});

  factory WorkflowJob.fromJson(Map<String, dynamic> json) => WorkflowJob(
    id: (json['id'] as num?)?.toInt() ?? 0,
    type: json['type'] as String? ?? 'Proceso',
    status: json['status'] as String? ?? 'pending',
    attempts: (json['attempts'] as num?)?.toInt() ?? 0,
    maxAttempts: (json['max_attempts'] as num?)?.toInt() ?? 1,
    lastError: json['last_error'] as String? ?? '',
  );
}


class DiskHealth {
  final String path;
  final double availableGb;
  final double totalGb;
  final double usedGb;

  const DiskHealth({required this.path, required this.availableGb, required this.totalGb, required this.usedGb});

  factory DiskHealth.fromJson(Map<String, dynamic> json) => DiskHealth(
    path: json['path'] as String? ?? '',
    availableGb: (json['available_gb'] as num?)?.toDouble() ?? 0,
    totalGb: (json['total_gb'] as num?)?.toDouble() ?? 0,
    usedGb: (json['used_gb'] as num?)?.toDouble() ?? 0,
  );
}
