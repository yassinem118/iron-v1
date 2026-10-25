import 'package:flutter/material.dart';
import 'package:url_launcher/url_launcher.dart';
import '../services/api_service.dart';

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

  // Supply the official Melbet URL verified for your deployment/jurisdiction:
  // flutter build apk --dart-define=MELBET_URL=https://your-verified-official-domain/
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
    setState(() { _busy = true; _error = null; });

    final result = await ApiService().login(_id.text.trim(), _password.text);
    if (!mounted) return;

    if (result != null) {
      setState(() { _busy = false; _error = result; });
      return;
    }

    // Use the official HTTPS URL. Android App Links / iOS Universal Links may
    // route it into the installed official app; otherwise the OS opens the web page.
    try {
      final uri = Uri.parse(_melbetUrl);
      if (uri.scheme != 'https' || uri.host.isEmpty) {
        throw const FormatException('MELBET_URL must be an HTTPS URL');
      }
      final launched = await launchUrl(uri, mode: LaunchMode.externalApplication);
      if (!mounted) return;
      setState(() => _busy = false);
      if (!launched) {
        setState(() => _error = 'Could not open the official destination. Please try again.');
      }
    } catch (_) {
      if (!mounted) return;
      setState(() {
        _busy = false;
        _error = 'Could not open the destination. Check the configured official URL.';
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    const accent = Color(0xFFFFD400); // اللون الأصفر المميز لـ Melbet
    const background = Color(0xFF0B0D12);
    return Scaffold(
      backgroundColor: background,
      body: SafeArea(
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
                    // تم تغيير الكلمة إلى MELBET باللون الأصفر الأنيق
                    const Text('MELBET', textAlign: TextAlign.center,
                      style: TextStyle(fontSize: 30, fontWeight: FontWeight.w800, letterSpacing: 3, color: accent)),
                    const SizedBox(height: 8),
                    const Text('Sign in to continue', textAlign: TextAlign.center,
                      style: TextStyle(color: Colors.white60)),
                    const SizedBox(height: 36),
                    const Text('ID', style: TextStyle(color: Colors.white70, fontWeight: FontWeight.w600)),
                    const SizedBox(height: 8),
                    TextFormField(
                      controller: _id,
                      autocorrect: false,
                      textInputAction: TextInputAction.next,
                      style: const TextStyle(color: Colors.white),
                      decoration: const InputDecoration(
                        hintText: 'User / License ID',
                        prefixIcon: Icon(Icons.person_outline),
                        border: OutlineInputBorder(),
                      ),
                      validator: (v) => v == null || v.trim().length < 3
                          ? 'Enter a valid ID' : null,
                    ),
                    const SizedBox(height: 18),
                    const Text('PASSWORD', style: TextStyle(color: Colors.white70, fontWeight: FontWeight.w600)),
                    const SizedBox(height: 8),
                    TextFormField(
                      controller: _password,
                      obscureText: _hide,
                      textInputAction: TextInputAction.done,
                      onFieldSubmitted: (_) => _submit(),
                      style: const TextStyle(color: Colors.white),
                      decoration: InputDecoration(
                        hintText: 'Your password',
                        prefixIcon: const Icon(Icons.lock_outline),
                        border: const OutlineInputBorder(),
                        suffixIcon: IconButton(
                          onPressed: () => setState(() => _hide = !_hide),
                          icon: Icon(_hide ? Icons.visibility : Icons.visibility_off),
                        ),
                      ),
                      validator: (v) => v == null || v.isEmpty ? 'Enter your password' : null,
                    ),
                    if (_error != null) ...[
                      const SizedBox(height: 14),
                      Text(_error!, style: const TextStyle(color: Colors.redAccent),
                        textAlign: TextAlign.center),
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
                          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
                        ),
                        child: _busy
                            ? const SizedBox(width: 22, height: 22,
                                child: CircularProgressIndicator(strokeWidth: 2, color: Colors.black))
                            : const Text('connect', style: TextStyle(fontSize: 16, fontWeight: FontWeight.w800)),
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
    );
  }
}