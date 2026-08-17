import 'package:flutter_test/flutter_test.dart';
import 'package:plezy/models/media_operations/media_operations_models.dart';

void main() {
  test('DiskHealth parses monitor metrics from the orchestrator', () {
    final health = DiskHealth.fromJson({
      'path': 'C:/Plex',
      'available_gb': 512.5,
      'total_gb': 1000,
      'used_gb': 487.5,
    });

    expect(health.path, 'C:/Plex');
    expect(health.availableGb, 512.5);
    expect(health.totalGb, 1000);
    expect(health.usedGb, 487.5);
  });
}
