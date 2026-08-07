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
