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
  bool _isLoading = false;
  List<dynamic> _rounds = [];
  String? _error;

  static const String _apiBaseUrl = 'https://iron-v1.vercel.app'; // رابط الباك إند

  // دالة طلب الـ Hack وجلب النتائج الحقيقية عند النقر على الزر
  Future<void> _fetchAppleFortuneResults() async {
    setState(() {
      _isLoading = true;
      _error = null;
    });

    try {
      final response = await http.get(Uri.parse('$_apiBaseUrl/api/apple-fortune/rounds'));
      
      if (!mounted) return;

      if (response.statusCode == 200) {
        final data = jsonDecode(response.body);
        setState(() {
          _rounds = data['rounds'] ?? [];
          _isLoading = false;
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
        _error = 'خطأ في الاتصال بالشبكة. تأكد من الإنترنت.';
      });
    }
  }

  // دالة فتح تطبيق Melbet عبر الـ Intent
  Future<void> _openMelbet() async {
    final Uri appUri = Uri.parse('intent://#Intent;package=com.melbet.client;end;');
    final Uri webUri = Uri.parse('https://melbet.com');
    
    try {
      if (await canLaunchUrl(appUri)) {
        await launchUrl(appUri, mode: LaunchMode.externalApplication);
      } else if (await canLaunchUrl(webUri)) {
        await launchUrl(webUri, mode: LaunchMode.externalApplication);
      }
    } catch (_) {
      await launchUrl(webUri, mode: LaunchMode.externalApplication);
    }
  }

  @override
  Widget build(BuildContext context) {
    const accentColor = Color(0xFFFF2A2A); // اللون الأحمر الدموي لهوية IRON

    return Scaffold(
      backgroundColor: const Color(0xFF080C12),
      appBar: AppBar(
        backgroundColor: const Color(0xFF111823),
        title: const Text('IRON // APPLE FORTUNE LIVE', style: TextStyle(color: accentColor, fontWeight: FontWeight.w900, fontSize: 16)),
        actions: [
          IconButton(
            icon: const Icon(Icons.open_in_new, color: Colors.white70),
            onPressed: _openMelbet,
            tooltip: 'فتح Melbet',
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
                Text(_rounds.isEmpty ? 'في انتظار طلب الـ Hack' : 'تم جلب النتائج بنجاح', style: const TextStyle(color: Colors.white38, fontSize: 11)),
              ],
            ),
          ),
          
          // عرض الشبكة: إما فارغة ومنتظمة في البداية، أو معمرة بالنتائج الحقيقية عند النقر
          Expanded(
            child: _rounds.isEmpty
                ? Center(
                    child: SingleChildScrollView(
                      padding: const EdgeInsets.all(16.0),
                      child: Column(
                        mainAxisAlignment: MainAxisAlignment.center,
                        children: [
                          const Text(
                            'اضغط على زر "HACK APPLE FORTUNE" أدناه\nبعد بدء الجولة في تطبيق Melbet',
                            textAlign: TextAlign.center,
                            style: TextStyle(color: Colors.white70, fontSize: 13, height: 1.5),
                          ),
                          const SizedBox(height: 25),
                          // عرض 10 صفوف فارغة مطابقة للعبة Apple Fortune (كل صف فيه 5 خانات دائرية)
                          ...List.generate(10, (rowIndex) {
                            return Container(
                              margin: const EdgeInsets.only(bottom: 10),
                              padding: const EdgeInsets.symmetric(vertical: 8, horizontal: 12),
                              decoration: BoxDecoration(
                                color: const Color(0xFF111823),
                                borderRadius: BorderRadius.circular(10),
                                border: Border.all(color: Colors.white12, width: 1),
                              ),
                              child: Row(
                                mainAxisAlignment: MainAxisAlignment.spaceEvenly,
                                children: List.generate(5, (colIndex) {
                                  return Container(
                                    width: 38,
                                    height: 38,
                                    decoration: BoxDecoration(
                                      color: const Color(0xFF1A2638),
                                      shape: BoxShape.circle,
                                      border: Border.all(color: Colors.white24, width: 1),
                                    ),
                                    child: Center(
                                      child: Text(
                                        '${colIndex + 1}',
                                        style: const TextStyle(color: Colors.white38, fontSize: 11),
                                      ),
                                    ),
                                  );
                                }),
                              ),
                            );
                          }),
                          if (_error != null) ...[
                            const SizedBox(height: 15),
                            Text(_error!, style: const TextStyle(color: Colors.redAccent), textAlign: TextAlign.center),
                          ],
                        ],
                      ),
                    ),
                  )
                : ListView.builder(
                    padding: const EdgeInsets.all(16),
                    itemCount: _rounds.length,
                    itemBuilder: (context, index) {
                      final round = _rounds[index];
                      final int safeColumn = round['safeColumn'] ?? -1;

                      return Container(
                        margin: const EdgeInsets.only(bottom: 12),
                        padding: const EdgeInsets.all(10),
                        decoration: BoxDecoration(
                          color: const Color(0xFF111823),
                          borderRadius: BorderRadius.circular(12),
                          border: Border.all(color: accentColor.withOpacity(0.4), width: 1),
                        ),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Row(
                              mainAxisAlignment: MainAxisAlignment.spaceBetween,
                              children: [
                                Text(
                                  'الجولة / الصف: ${round['roundId'] ?? index + 1}',
                                  style: const TextStyle(color: Colors.white, fontWeight: FontWeight.bold, fontSize: 12),
                                ),
                                const Text('LIVE SIGNAL', style: TextStyle(color: Colors.green, fontSize: 10, fontWeight: FontWeight.bold)),
                              ],
                            ),
                            const SizedBox(height: 8),
                            // عرض 5 خانات لكل صف وتلوين الخانة الآمنة الحقيقية القادمة من السيرفر
                            Row(
                              mainAxisAlignment: MainAxisAlignment.spaceEvenly,
                              children: List.generate(5, (colIndex) {
                                final bool isSafe = colIndex == safeColumn;
                                return Container(
                                  width: 42,
                                  height: 42,
                                  decoration: BoxDecoration(
                                    color: isSafe ? accentColor.withOpacity(0.3) : const Color(0xFF1A2638),
                                    shape: BoxShape.circle,
                                    border: Border.all(
                                      color: isSafe ? accentColor : Colors.white24,
                                      width: isSafe ? 2 : 1,
                                    ),
                                  ),
                                  child: Center(
                                    child: isSafe
                                        ? const Icon(Icons.star, color: accentColor, size: 20)
                                        : Text(
                                            '${colIndex + 1}',
                                            style: const TextStyle(color: Colors.white38, fontSize: 11),
                                          ),
                                  ),
                                );
                              }),
                            ),
                          ],
                        ),
                      );
                    },
                  ),
          ),

          // زر HACK APPLE FORTUNE في الأسفل للتحكم اليدوي بعد اللعب
          Padding(
            padding: const EdgeInsets.all(16.0),
            child: SizedBox(
              width: double.infinity,
              height: 52,
              child: FilledButton(
                style: FilledButton.styleFrom(
                  backgroundColor: accentColor,
                  foregroundColor: Colors.white,
                  shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
                ),
                onPressed: _isLoading ? null : _fetchAppleFortuneResults,
                child: _isLoading
                    ? const SizedBox(
                        width: 22,
                        height: 22,
                        child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white),
                      )
                    : const Text(
                        'HACK APPLE FORTUNE',
                        style: TextStyle(fontSize: 15, fontWeight: FontWeight.w900, letterSpacing: 1.5),
                      ),
              ),
            ),
          ),
        ],
      ),
    );
  }
}