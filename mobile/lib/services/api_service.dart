import 'dart:convert';
import 'package:http/http.dart' as http;

class ApiService {
  static const baseUrl = String.fromEnvironment(
    'IRON_API_URL',
    defaultValue: 'http://10.0.2.2:8080',
  );

  Future<String?> login(String accountId, String password) async {
    try {
      final uri = Uri.parse('$baseUrl/api/auth/login');
      final response = await http
          .post(
            uri,
            headers: {'content-type': 'application/json'},
            body: jsonEncode({'accountId': accountId, 'password': password}),
          )
          .timeout(const Duration(seconds: 15));

      if (response.statusCode != 200) {
        try {
          return jsonDecode(response.body)['error'] as String? ??
              'Invalid ID or password';
        } catch (_) {
          return 'Login failed. Please try again.';
        }
      }
      return null;
    } catch (_) {
      return 'Could not reach the server. Check your connection and try again.';
    }
  }
}
