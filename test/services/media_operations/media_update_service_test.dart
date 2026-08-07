import 'package:flutter_test/flutter_test.dart';
import 'package:plezy/services/media_operations/media_update_service.dart';

void main() {
  test('compares upstream version, custom revision and Android build number', () {
    expect(MediaUpdateService.isNewerForTest('2.12.1-2+2005', '2.12.1-1+2004'), isTrue);
    expect(MediaUpdateService.isNewerForTest('2.12.2-1+2004', '2.12.1-9+2999'), isTrue);
    expect(MediaUpdateService.isNewerForTest('2.12.1-1+2004', '2.12.1-1+2004'), isFalse);
    expect(MediaUpdateService.isNewerForTest('2.12.1-1+2003', '2.12.1-1+2004'), isFalse);
  });

  test('preserves installer state for the Android update flow', () {
    const result = ApkInstallResult(started: false, requiresPermission: true);
    expect(result.started, isFalse);
    expect(result.requiresPermission, isTrue);
  });
}
