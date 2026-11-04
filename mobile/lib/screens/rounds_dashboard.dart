import 'dart:async';
import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;
import 'package:url_launcher/url_launcher.dart';

class RoundsDashboard extends StatefulWidget {
  const RoundsDashboard({super.key});

  @override
  State<RoundsDashboard> createState() => _RoundsDashboardState();
}

class _RoundsDashboardState extends State<RoundsDashboard> {
  bool _isLoading = true;
  List<dynamic> _rounds = [];
  String? _error;
  Timer? _timer;

  static const String _apiBaseUrl = 'https://iron-v1.vercel.app'; // رابط الباك إند

  @override
  void initState() {
    super.initState();
    _fetchAppleFortuneResults();
    // جلب النتائج تلقائياً كل 5 ثوانٍ لتحديث مباشر (Live)
    _timer = Timer.periodic(const Duration(seconds: 5), (_) => _fetchAppleFortuneResults());
  }

  @override
  void dispose() {
    _timer?.cancel();
    super.dispose();
  }

  // جلب النتائج الخاصة بلعبة Apple Fortune من مسار الـ Backend
  Future<void> _fetchAppleFortuneResults() async {
    try {
      final response = await http.get(Uri.parse('$_apiBaseUrl/api/apple-fortune/rounds'));
      
      if (!mounted) return;

      if (response.statusCode == 200) {
        final data = jsonDecode(response.body);
        setState(() {
          _rounds = data['rounds'] ?? [];
          _isLoading = false;
          _error = null;
        });
      } else {
        setState(() {
          _isLoading = false;
          _error = 'تعذر الاتصال بسيرفر النتائج المؤكدة.';
        });
      }
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _isLoading = false;
        _error = 'خطأ في الاتصال بالشبكة.';
      });
    }
  }

  // دالة فتح تطبيق Melbet مباشرة
  Future<void> _openMelbet() async {
    final Uri appUri = Uri.parse('melbet://');
    final Uri webUri = Uri.parse('https://melbet.com');
    try {
      if (await canLaunchUrl(appUri)) {
        await launchUrl(appUri, mode: LaunchMode.externalApplication);
      } else {
        await launchUrl(webUri, mode: LaunchMode.externalApplication);
      }
    } catch (_) {
      await launchUrl(webUri, mode: LaunchMode.externalApplication);
    }
  }

  @override
  Widget build(BuildContext context) {
    const accentColor = Color(0xFFFF2A2A); // أحمر دموي تناسقاً مع الهوية الجديدة

    return Scaffold(
      backgroundColor: const Color(0xFF080C12),
      appBar: AppBar(
        backgroundColor: const Color(0xFF111823),
        title: const Text('IRON // APPLE FORTUNE LIVE', style: TextStyle(color: accentColor, fontWeight: FontWeight.w900, fontSize: 16)),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh, color: Colors.white70),
            onPressed: () {
              setState(() => _isLoading = true);
              _fetchAppleFortuneResults();
            },
          ),
        ],
      ),
      body: Column(
        children: [
          // شريط الحالة الحي
          Container(
            padding: const EdgeInsets.all(12),
            color: const Color(0xFF161F30),
            child: Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                const Row(
                  children: [
                    CircleAvatar(radius: 5, backgroundColor: Colors.green),
                    SizedBox(width: 8),
                    Text('البوت متصل ويفك التشفير', style: TextStyle(color: Colors.white70, fontSize: 12)),
                  ],
                ),
                Text('تحديث كل 5 ثوانٍ', style: TextStyle(color: Colors.white38, fontSize: 11)),
              ],
            ),
          ),
          
          // عرض النتائج القادمة من البوت
          Expanded(
            child: _isLoading
                ? const Center(child: CircularProgressIndicator(color: accentColor))
                : _error != null
                    ? Center(
                        child: Text(_error!, style: const TextStyle(color: Colors.redAccent), textAlign: TextAlign.center),
                      )
                    _rounds.isEmpty
    ? const Center(
        child: Text('لا توجد جولات متاحة حالياً. في انتظار بيانات البوت...', 
        style: TextStyle(color: Colors.white54), textAlign: TextAlign.center),
      )
                        : ListView.builder(
                            padding: const EdgeInsets.all(16),
                            itemCount: _rounds.length,
                            itemBuilder: (context, index) {
                              final round = _rounds[index];
                              return Card(
                                color: const Color(0xFF111823),
                                shape: RoundedRectangleBorder(
                                  borderRadius: BorderRadius.circular(12),
                                  side: const BorderSide(color: Colors.redAccent, width: 0.5),
                                ),
                                margin: const EdgeInsets.only(bottom: 12),
                                child: ListTile(
                                  leading: const Icon(Icons.security, color: accentColor),
                                  title: Text('جولة رقم: ${round['roundId'] ?? index + 1}', 
                                      style: const TextStyle(color: Colors.white, fontWeight: FontWeight.bold)),
                                  subtitle: Text('النتيجة المؤكدة: ${round['prediction'] ?? 'جاري التحليل...'}', 
                                      style: const TextStyle(color: Colors.white70)),
                                  trailing: const Text('LIVE', style: TextStyle(color: Colors.green, fontWeight: FontWeight.bold)),
                                ),
                              );
                            },
                          ),
          ),

          // زر فتح Melbet السريع في الأسفل
          Padding(
            padding: const EdgeInsets.all(16.0),
            child: SizedBox(
              width: double.infinity,
              height: 50,
              child: ElevatedButton.icon(
                style: ElevatedButton.styleFrom(
                  backgroundColor: accentColor,
                  foregroundColor: Colors.white,
                  shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
                ),
                onPressed: _openMelbet,
                icon: const Icon(Icons.open_in_new),
                label: const Text('فتح تطبيق Melbet للعب', style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold)),
              ),
            ),
          ),
        ],
      ),
    );
  }
}