// GGID SAML SSO Demo with Permissions (Dart)
// Run: GGID_URL=... dart run saml_permission_demo.dart
import 'dart:io';
import '../../lib/src/saml.dart';

class DemoUser {
  final String username;
  final List<String> roles;
  final List<String> permissions;
  DemoUser(this.username, this.roles, this.permissions);
  bool hasPermission(String perm) =>
      permissions.contains('admin') || permissions.contains(perm);
}

void main() async {
  final ggidUrl = Platform.environment['GGID_URL'] ?? 'http://localhost:8080';
  final entityId = Platform.environment['SP_ENTITY_ID'] ?? 'http://localhost:3101/saml/metadata';
  final acsUrl = Platform.environment['ACS_URL'] ?? 'http://localhost:3101/saml/acs';
  final currentUser = DemoUser('demo_user', ['viewer'], ['dashboard:read', 'orders:read', 'inventory:read']);

  final server = await HttpServer.bind(InternetAddress.loopbackIPv4, 3101);
  print('SAML Permission Demo on http://localhost:3101');

  await for (final request in server) {
    final path = request.uri.path;
    final response = request.response;

    if (path == '/') {
      response.headers.contentType = ContentType.html;
      response.write(renderDashboard(currentUser));
    } else if (path == '/saml/metadata') {
      response.headers.contentType = ContentType.parse('application/xml');
      response.write(generateSPMetadata(SAMLConfig(entityId: entityId, acsUrl: acsUrl)));
    } else if (path == '/login') {
      final ssoUrl = '$ggidUrl/saml/sso';
      final url = buildAuthnRequestUrl(ssoUrl: ssoUrl, entityId: entityId, acsUrl: acsUrl);
      response.statusCode = HttpStatus.movedPermanently;
      response.headers.set(HttpHeaders.locationHeader, url);
    } else if (path == '/inventory') {
      response.headers.contentType = ContentType.html;
      if (!currentUser.hasPermission('inventory:read')) {
        response.statusCode = HttpStatus.forbidden;
        response.write(render403('inventory:read'));
      } else {
        response.write(renderPage('Inventory', currentUser,
          canWrite: currentUser.hasPermission('inventory:write')));
      }
    } else if (path == '/orders') {
      response.headers.contentType = ContentType.html;
      if (!currentUser.hasPermission('orders:read')) {
        response.statusCode = HttpStatus.forbidden;
        response.write(render403('orders:read'));
      } else {
        response.write(renderPage('Orders', currentUser,
          canWrite: currentUser.hasPermission('orders:write'),
          canApprove: currentUser.hasPermission('orders:approve')));
      }
    } else {
      response.statusCode = HttpStatus.notFound;
      response.write('Not found');
    }
    await response.close();
  }
}

String renderMenu(DemoUser user) {
  final items = ['<li><a href="/">Dashboard</a></li>'];
  if (user.hasPermission('orders:read')) items.add('<li><a href="/orders">Orders</a></li>');
  if (user.hasPermission('inventory:read')) items.add('<li><a href="/inventory">Inventory</a></li>');
  return '<aside><h2>Menu</h2><ul>${items.join()}</ul><p>Roles: ${user.roles.join(', ')}</p></aside>';
}

String renderDashboard(DemoUser user) => '<html><body>$renderMenu<main><h1>Dashboard</h1><p>Welcome ${user.username}</p></main></body></html>'
  .replaceAll('renderMenu', renderMenu(user));

String renderPage(String title, DemoUser user, {bool canWrite = false, bool canApprove = false}) {
  final buttons = [if (canWrite) '<button>New</button>', if (canApprove) '<button>Approve</button>'].join(' ');
  return '<html><body>${renderMenu(user)}<main><h1>$title</h1>$buttons</main></body></html>';
}

String render403(String perm) => '<html><body><h1>403</h1><p>Need: $perm</p></body></html>';
