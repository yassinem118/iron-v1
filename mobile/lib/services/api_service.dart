import 'dart:convert';
import 'package:http/http.dart' as http;

class ApiService {
  static const baseUrl = String.fromEnvironment(
    'IRON_API_URL',
    defaultValue: 'https://iron-v1.vercel.app',
  );

  // دالة الاتصال الجديدة اللي كتتوافق مع مسار /api/connect
  Future<String?> connect(String accountId, String password) async {
    try {
      final uri = Uri.parse('$baseUrl/api/connect');
      final response = await http
          .post(
            uri,
            headers: {'content-type': 'application/json'},
            body: jsonEncode({'accountId': accountId, 'password': password}),
          )
          .timeout(const Duration(seconds: 15));

      final data = jsonDecode(response.body);

      if (response.statusCode == 200 && data['success'] == true) {
        return null; // نجاح العملية (ما كاين حتى خطأ)
      } else {
        return data['message'] ?? 'Invalid ID or password';
      }
    } catch (_) {
      return 'Could not reach the server. Check your connection and try again.';
    }
  }
}