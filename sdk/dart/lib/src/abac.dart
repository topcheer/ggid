/// ABAC operations for the GGID Dart SDK.
///
/// The core ABAC methods (evaluateAbac, checkPolicy) are already available
/// as instance methods on [GGIDClient].
///
/// This file provides extension methods for additional ABAC utilities.
library ggid.abac;

import 'package:ggid/ggid_client.dart';
import 'models.dart';

/// Extension providing convenience ABAC operations.
extension AbacExtension on GGIDClient {
  /// Convenience: check policy with simple parameters.
  ///
  /// ```dart
  /// final allowed = await ggid.checkPolicySimple(
  ///   token, 'user-1', 'documents', 'read',
  ///   context: {'department': 'finance'},
  /// );
  /// ```
  Future<bool> checkPolicySimple(
    String token,
    String subject,
    String resource,
    String action, {
    Map<String, String> context = const {},
  }) async {
    final result = await checkPolicy(token, PolicyCheckRequest(
      subject: subject,
      resource: resource,
      action: action,
      context: context,
    ));
    return result.allowed;
  }

  /// Evaluate ABAC with a simple condition list.
  Future<AbacEvalResult> evaluateAbacSimple(
    String token,
    String resource,
    String action,
    List<AbacCondition> conditions,
  ) async {
    return evaluateAbac(token, AbacEvalRequest(
      action: action,
      resource: resource,
      conditions: conditions,
    ));
  }
}
