import 'dart:convert';
import 'package:http/http.dart' as http;

class ApiService {
  static const String baseUrl = String.fromEnvironment(
    'IRON_API_URL',
    defaultValue: 'https://iron-v1.vercel.app',
  );

  // دالة الاتصال المحدثة ترجع Map تحتوي على حالة النجاح ورسالة الخطأ إن وجدات
  static Future<Map<String, dynamic>> connect(String accountId, String password) async {
    try {
      final uri = Uri.parse('$baseUrl/api/connect');
      final response = await http
          .post(
            uri,
            headers: {'Content-Type': 'application/json'},
            body: jsonEncode({
              'accountId': accountId.trim(),
              'password': password,
            }),
          )
          .timeout(const Duration(seconds: 15));

      final data = jsonDecode(response.body);

      if (response.statusCode == 200 && data['success'] == true) {
        return {'success': true};
      } else {
        return {
          'success': false,
          'message': data['message'] ?? 'Invalid ID or password'
        };
      }
    } catch (e) {
      return {
        'success': false,
        'message': 'Connection failed: ${e.toString()}'
      };
    }
  }
}