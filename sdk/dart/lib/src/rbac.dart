/// RBAC operations for the GGID Dart SDK.
///
/// The core RBAC methods (checkPermission, assignRole, revokeRole, getUserRoles,
/// listRoles, listPermissions, createRole, deleteRole) are already available
/// as instance methods on [GGIDClient].
///
/// This file provides extension methods for additional RBAC utilities.
library ggid.rbac;

import 'package:ggid/ggid_client.dart';
import 'models.dart';

/// Extension providing convenience RBAC operations.
extension RbacExtension on GGIDClient {
  /// Check if a user has a specific role.
  ///
  /// ```dart
  /// final isAdmin = await ggid.hasRole(token, 'user-1', 'admin');
  /// ```
  Future<bool> hasRole(String token, String userId, String roleKey) async {
    final roles = await getUserRoles(token, userId);
    return roles.any((r) => r.key == roleKey);
  }

  /// Check if a user has any of the specified roles.
  Future<bool> hasAnyRole(String token, String userId, List<String> roleKeys) async {
    final roles = await getUserRoles(token, userId);
    final userRoleKeys = roles.map((r) => r.key).toSet();
    return roleKeys.any(userRoleKeys.contains);
  }

  /// Find a role by its key.
  Future<Role?> findRoleByKey(String token, String key) async {
    final roles = await listRoles(token);
    for (final r in roles) {
      if (r.key == key) return r;
    }
    return null;
  }
}
