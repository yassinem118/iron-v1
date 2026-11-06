import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:url_launcher/url_launcher.dart';
import 'rounds_dashboard.dart';

class LoginScreen extends StatefulWidget {
  const LoginScreen({super.key});

  @override
  State<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends State<LoginScreen> {
  final _form = GlobalKey<FormState>();
  final _id = TextEditingController();
  final _password = TextEditingController();
  bool _busy = false;
  bool _hide = true;
  String? _error;

  static const Color bloodRed = Color(0xFF990000);
  static const Color accent = Color(0xFFFF2A2A);

  @override
  void dispose() {
    _id.dispose();
    _password.dispose();
    super.dispose();
  }

  // فتح تطبيق Melbet مباشرة وإجبار الأندرويد على إطلاق الحزمة بدون متصفح
  Future<void> _openMelbetAppDirectly() async {
    final Uri appUri = Uri.parse('intent://#Intent;package=org.melbet.vac;end;');
    
    try {
      if (await canLaunchUrl(appUri)) {
        await launchUrl(appUri, mode: LaunchMode.externalApplication);
      } else {
        final Uri fallbackUri = Uri.parse('package:org.melbet.vac');
        await launchUrl(fallbackUri, mode: LaunchMode.externalApplication);
      }
    } catch (_) {}
  }

  // دالة الـ submit المطورة مع إشعار نجاح الاتصال الفوري
  Future<void> _submit() async {
    if (_busy || !_form.currentState!.validate()) return;
    setState(() {
      _busy = true;
      _error = null;
    });

    // إظهار إشعار سريع للمستخدم بلي الاتصال تم بنجاح
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(
        content: Text('Connected successfully! Launching Melbet...'),
        backgroundColor: Colors.green,
        duration: Duration(milliseconds: 1500),
      ),
    );

    // انتظار لحظة قصيرة باش يبان الإشعار، عاد فتح تطبيق Melbet
    await Future.delayed(const Duration(milliseconds: 800));
    await _openMelbetAppDirectly();

    if (!mounted) return;
    setState(() => _busy = false);

    // الانتقال للداشبورد بسلاسة
    Navigator.of(context).pushReplacement(
      MaterialPageRoute(builder: (_) => const RoundsDashboard()),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Stack(
        children: [
          Positioned.fill(
            child: DecoratedBox(
              decoration: const BoxDecoration(
                gradient: LinearGradient(
                  begin: Alignment.topLeft,
                  end: Alignment.bottomRight,
                  colors: [Color(0xFF161F30), Color(0xFF080C12), Color(0xFF080C12)],
                ),
              ),
            ),
          ),
          SafeArea(
            child: Center(
              child: SingleChildScrollView(
                padding: const EdgeInsets.all(24),
                child: ConstrainedBox(
                  constraints: const BoxConstraints(maxWidth: 420),
                  child: Form(
                    key: _form,
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        // عرض اللوغو الحقيقي مباشرة بدون إطار أحمر فارغ
                        Center(
                          child: Container(
                            width: 130,
                            height: 130,
                            decoration: BoxDecoration(
                              shape: BoxShape.circle,
                              boxShadow: [
                                BoxShadow(
                                  color: bloodRed.withOpacity(0.4),
                                  blurRadius: 25,
                                  spreadRadius: 5,
                                ),
                              ],
                              image: const DecorationImage(
                                image: AssetImage('assets/icon/iron_logo.jpg'), // أو .png حسب ملفك
                                fit: BoxFit.cover,
                              ),
                            ),
                          ),
                        ),
                        const SizedBox(height: 18),
                        const Text(
                          'IRON',
                          textAlign: TextAlign.center,
                          style: TextStyle(
                            fontSize: 34,
                            fontWeight: FontWeight.w900,
                            letterSpacing: 3,
                            color: bloodRed,
                          ),
                        ),
                        const SizedBox(height: 6),
                        const Text(
                          'Sign in to access your IRON dashboard',
                          textAlign: TextAlign.center,
                          style: TextStyle(color: Colors.white70, fontSize: 13),
                        ),
                        const SizedBox(height: 36),
                        const Text(
                          'EMAIL / ID',
                          style: TextStyle(color: Colors.white70, fontWeight: FontWeight.w600, fontSize: 12),
                        ),
                        const SizedBox(height: 8),
                        TextFormField(
                          controller: _id,
                          autocorrect: false,
                          textInputAction: TextInputAction.next,
                          style: const TextStyle(color: Colors.white),
                          decoration: InputDecoration(
                            hintText: 'Enter your Email or User ID',
                            hintStyle: const TextStyle(color: Colors.white38),
                            prefixIcon: const Icon(Icons.person_outline, color: bloodRed),
                            filled: true,
                            fillColor: const Color(0xFF111823),
                            border: OutlineInputBorder(borderRadius: BorderRadius.circular(12)),
                            enabledBorder: OutlineInputBorder(
                              borderRadius: BorderRadius.circular(12),
                              borderSide: const BorderSide(color: Colors.white24),
                            ),
                            focusedBorder: OutlineInputBorder(
                              borderRadius: BorderRadius.circular(12),
                              borderSide: const BorderSide(color: bloodRed, width: 2),
                            ),
                          ),
                          validator: (v) => v == null || v.trim().isEmpty
                              ? 'Enter a valid Email or ID' : null,
                        ),
                        const SizedBox(height: 18),
                        const Text(
                          'MOT PASSE',
                          style: TextStyle(color: Colors.white70, fontWeight: FontWeight.w600, fontSize: 12),
                        ),
                        const SizedBox(height: 8),
                        TextFormField(
                          controller: _password,
                          obscureText: _hide,
                          textInputAction: TextInputAction.done,
                          onFieldSubmitted: (_) => _submit(),
                          style: const TextStyle(color: Colors.white),
                          decoration: InputDecoration(
                            hintText: 'Enter your password',
                            hintStyle: const TextStyle(color: Colors.white38),
                            prefixIcon: const Icon(Icons.lock_outline, color: bloodRed),
                            filled: true,
                            fillColor: const Color(0xFF111823),
                            border: OutlineInputBorder(borderRadius: BorderRadius.circular(12)),
                            enabledBorder: OutlineInputBorder(
                              borderRadius: BorderRadius.circular(12),
                              borderSide: const BorderSide(color: Colors.white24),
                            ),
                            focusedBorder: OutlineInputBorder(
                              borderRadius: BorderRadius.circular(12),
                              borderSide: const BorderSide(color: bloodRed, width: 2),
                            ),
                            suffixIcon: IconButton(
                              onPressed: () => setState(() => _hide = !_hide),
                              icon: Icon(_hide ? Icons.visibility : Icons.visibility_off, color: Colors.white70),
                            ),
                          ),
                          validator: (v) => v == null || v.isEmpty ? 'Enter your password' : null,
                        ),
                        if (_error != null) ...[
                          const SizedBox(height: 14),
                          Text(
                            _error!,
                            style: const TextStyle(color: Colors.redAccent),
                            textAlign: TextAlign.center,
                          ),
                        ],
                        const SizedBox(height: 28),
                        SizedBox(
                          height: 54,
                          child: FilledButton(
                            onPressed: _busy ? null : _submit,
                            style: FilledButton.styleFrom(
                              backgroundColor: bloodRed,
                              foregroundColor: Colors.white,
                              disabledBackgroundColor: bloodRed.withOpacity(0.55),
                              shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
                            ),
                            child: _busy
                                ? const SizedBox(
                                    width: 22,
                                    height: 22,
                                    child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white),
                                  )
                                : const Text(
                                    'CONNECT',
                                    style: TextStyle(fontSize: 16, fontWeight: FontWeight.w900, letterSpacing: 1.5),
                                  ),
                          ),
                        ),
                      ],
                    ),
                  ),
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }
}