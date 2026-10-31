import 'dart:async';
import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;
import 'package:url_launcher/url_launcher.dart';

const _accent = Color(0xFFFFD400);
const _panel = Color(0xFF111A29);

class RoundRecord {
  final String id;
  final int round;
  final int row;
  final int cell;
  final DateTime? time;
  final String status;

  const RoundRecord({required this.id, required this.round, required this.row,
    required this.cell, required this.status, this.time});

  factory RoundRecord.fromJson(Map<String, dynamic> json) {
    int number(dynamic value, int fallback) => value is num ? value.toInt() : int.tryParse('$value') ?? fallback;
    final rawTime = json['createdAt'] ?? json['timestamp'] ?? json['time'];
    return RoundRecord(
      id: '${json['id'] ?? json['_id'] ?? '${json['round']}-${json['row']}-${json['cell']}'}',
      round: number(json['round'] ?? json['roundNumber'], 0),
      row: number(json['row'] ?? json['level'], 0),
      cell: number(json['cell'] ?? json['column'], 0),
      status: '${json['status'] ?? 'confirmed'}',
      time: rawTime == null ? null : DateTime.tryParse('$rawTime')?.toLocal(),
    );
  }
}

class RoundsDashboard extends StatefulWidget {
  const RoundsDashboard({super.key});
  @override
  State<RoundsDashboard> createState() => _RoundsDashboardState();
}

class _RoundsDashboardState extends State<RoundsDashboard> with WidgetsBindingObserver {
  static const _baseUrl = String.fromEnvironment('IRON_API_URL', defaultValue: 'http://78.154.103.8:16673');
  static const _roundsPath = String.fromEnvironment('IRON_ROUNDS_PATH', defaultValue: '/api/rounds');
  static const _melbetUrl = String.fromEnvironment('MELBET_URL', defaultValue: 'https://melbet.com/');
  final List<RoundRecord> _rounds = [];
  Timer? _timer;
  bool _loading = true;
  bool _opening = false;
  String? _error;
  DateTime? _lastUpdated;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _refresh();
    _timer = Timer.periodic(const Duration(seconds: 5), (_) => _refresh());
    // Preserve the existing flow: open the configured Melbet destination after successful IRON login.
    WidgetsBinding.instance.addPostFrameCallback((_) => _openMelbet(silent: true));
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed) _refresh();
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _timer?.cancel();
    super.dispose();
  }

  Future<void> _openMelbet({bool silent = false}) async {
    if (_opening) return;
    _opening = true;
    try {
      final uri = Uri.parse(_melbetUrl);
      final ok = await launchUrl(uri, mode: LaunchMode.externalApplication);
      if (!ok && mounted && !silent) _notice('ما قدرناش نفتح الرابط. راجع MELBET_URL.');
    } catch (_) {
      if (mounted && !silent) _notice('تعذر فتح الرابط.');
    } finally {
      _opening = false;
    }
  }

  Future<void> _refresh() async {
    try {
      final uri = Uri.parse('$_baseUrl$_roundsPath');
      final response = await http.get(uri, headers: {'Accept': 'application/json'}).timeout(const Duration(seconds: 8));
      if (response.statusCode != 200) throw Exception('HTTP ${response.statusCode}');
      final decoded = jsonDecode(response.body);
      final dynamic raw = decoded is List ? decoded : (decoded is Map ? (decoded['rounds'] ?? decoded['results'] ?? decoded['data'] ?? []) : []);
      if (raw is! List) throw const FormatException('Expected a list of round records');
      final parsed = raw.whereType<Map>().map((e) => RoundRecord.fromJson(Map<String, dynamic>.from(e)))
          .where((r) => r.round > 0 && r.row > 0 && r.cell > 0).toList()
        ..sort((a, b) => b.round.compareTo(a.round));
      if (!mounted) return;
      setState(() { _rounds..clear()..addAll(parsed); _loading = false; _error = null; _lastUpdated = DateTime.now(); });
    } catch (_) {
      if (!mounted) return;
      setState(() { _loading = false; _error = 'ما قدرناش نتاصلو بالـBackend ديال النتائج. تأكد من API والمسار.'; });
    }
  }

  void _notice(String message) => ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(message)));

  String _clock(DateTime? time) {
    if (time == null) return 'الوقت غير متوفر';
    final h = time.hour.toString().padLeft(2, '0');
    final m = time.minute.toString().padLeft(2, '0');
    final s = time.second.toString().padLeft(2, '0');
    return '$h:$m:$s';
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(
      title: const Row(children: [Icon(Icons.shield_moon, color: _accent), SizedBox(width: 10), Text('IRON', style: TextStyle(fontWeight: FontWeight.w900, letterSpacing: 2))]),
      actions: [IconButton(tooltip: 'تحديث النتائج', onPressed: _refresh, icon: const Icon(Icons.refresh)), const SizedBox(width: 8)],
    ),
    body: RefreshIndicator(
      onRefresh: _refresh,
      color: _accent,
      child: ListView(
        padding: const EdgeInsets.fromLTRB(18, 10, 18, 28),
        children: [
          Container(
            padding: const EdgeInsets.all(20),
            decoration: BoxDecoration(gradient: const LinearGradient(colors: [Color(0xFF202C40), Color(0xFF111A29)], begin: Alignment.topLeft, end: Alignment.bottomRight), borderRadius: BorderRadius.circular(22), border: Border.all(color: Colors.white.withOpacity(.08))),
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Row(children: [Container(width: 10, height: 10, decoration: const BoxDecoration(color: Color(0xFF35D39A), shape: BoxShape.circle)), const SizedBox(width: 8), const Text('لوحة الجولات', style: TextStyle(color: Colors.white70, fontSize: 13)), const Spacer(), _statusPill()],),
              const SizedBox(height: 18),
              const Text('نتائج الجولات', style: TextStyle(fontSize: 27, fontWeight: FontWeight.w900)),
              const SizedBox(height: 7),
              Text('النتائج المؤكدة اللي كيوصل بها الـBackend من المصدر المصرح به.', style: TextStyle(color: Colors.white.withOpacity(.62), height: 1.4)),
              const SizedBox(height: 20),
              Row(children: [Expanded(child: _metric('الجولات', '${_rounds.length}', Icons.grid_view_rounded)), const SizedBox(width: 12), Expanded(child: _metric('آخر تحديث', _lastUpdated == null ? '--:--' : _clock(_lastUpdated), Icons.sync_rounded))]),
            ]),
          ),
          const SizedBox(height: 22),
          Row(children: [const Expanded(child: Text('سجل الجولات', style: TextStyle(fontSize: 19, fontWeight: FontWeight.w800))), Text('تحديث كل 5 ثواني', style: TextStyle(fontSize: 11, color: Colors.white.withOpacity(.45)))]),
          const SizedBox(height: 12),
          if (_loading && _rounds.isEmpty) const Padding(padding: EdgeInsets.all(30), child: Center(child: CircularProgressIndicator(color: _accent))),
          if (_error != null) Container(padding: const EdgeInsets.all(14), decoration: BoxDecoration(color: const Color(0xFF301D24), borderRadius: BorderRadius.circular(14), border: Border.all(color: Colors.redAccent.withOpacity(.25))), child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [const Row(children: [Icon(Icons.cloud_off, color: Colors.redAccent), SizedBox(width: 8), Text('النتائج غير متاحة', style: TextStyle(fontWeight: FontWeight.bold))]), const SizedBox(height: 8), Text(_error!, style: const TextStyle(color: Colors.white70)), const SizedBox(height: 10), Align(alignment: Alignment.centerRight, child: TextButton(onPressed: _refresh, child: const Text('عاود المحاولة')))])),
          if (!_loading && _error == null && _rounds.isEmpty) Container(padding: const EdgeInsets.all(24), decoration: BoxDecoration(color: _panel, borderRadius: BorderRadius.circular(16)), child: const Column(children: [Icon(Icons.hourglass_empty, size: 34, color: _accent), SizedBox(height: 10), Text('ما وصلات حتى جولة', style: TextStyle(fontWeight: FontWeight.bold, fontSize: 16)), SizedBox(height: 6), Text('ملي الـBackend يتوصل بنتيجة مؤكدة ويرجعها من API غادي تبان هنا.', textAlign: TextAlign.center, style: TextStyle(color: Colors.white54, height: 1.4))])),
          ..._rounds.map(_roundCard),
          const SizedBox(height: 20),
          OutlinedButton.icon(onPressed: _opening ? null : () => _openMelbet(), icon: const Icon(Icons.open_in_new), label: const Text('فتح Melbet'), style: OutlinedButton.styleFrom(foregroundColor: _accent, side: const BorderSide(color: _accent), padding: const EdgeInsets.symmetric(vertical: 15))),
          const SizedBox(height: 12),
          Text('تنبيه: السجل كيعرض غير البيانات اللي رجعها السيرفر؛ ماشي توقع للخانات المخفية أو ضمان لنتائج مستقبلية.', textAlign: TextAlign.center, style: TextStyle(fontSize: 11, color: Colors.white.withOpacity(.42), height: 1.5)),
        ],
      ),
    ),
  );

  Widget _statusPill() => Container(padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6), decoration: BoxDecoration(color: const Color(0xFF17352F), borderRadius: BorderRadius.circular(30)), child: const Text('LIVE', style: TextStyle(color: Color(0xFF59E2B0), fontSize: 10, fontWeight: FontWeight.w900, letterSpacing: 1.2)));

  Widget _metric(String title, String value, IconData icon) => Container(padding: const EdgeInsets.all(13), decoration: BoxDecoration(color: Colors.black.withOpacity(.18), borderRadius: BorderRadius.circular(14)), child: Row(children: [Icon(icon, color: _accent, size: 20), const SizedBox(width: 10), Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [Text(title, style: TextStyle(fontSize: 11, color: Colors.white.withOpacity(.5)), maxLines: 1), const SizedBox(height: 4), Text(value, style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w800), maxLines: 1, overflow: TextOverflow.ellipsis)]))]));

  Widget _roundCard(RoundRecord r) => Container(
    margin: const EdgeInsets.only(bottom: 10), padding: const EdgeInsets.all(15),
    decoration: BoxDecoration(color: _panel, borderRadius: BorderRadius.circular(16), border: Border.all(color: Colors.white.withOpacity(.06))),
    child: Row(children: [
      Container(width: 46, height: 46, decoration: BoxDecoration(color: _accent.withOpacity(.12), borderRadius: BorderRadius.circular(13)), child: const Icon(Icons.bolt_rounded, color: _accent)),
      const SizedBox(width: 13),
      Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [Text('الجولة ${r.round}', style: const TextStyle(fontWeight: FontWeight.w800, fontSize: 16)), const SizedBox(height: 5), Text('الصف ${r.row}  •  الخانة ${r.cell}', style: const TextStyle(color: Colors.white70, fontSize: 13)), const SizedBox(height: 5), Text(_clock(r.time), style: TextStyle(color: Colors.white.withOpacity(.4), fontSize: 11))])),
      Container(padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 7), decoration: BoxDecoration(color: const Color(0xFF17352F), borderRadius: BorderRadius.circular(10)), child: const Text('مؤكدة', style: TextStyle(color: Color(0xFF59E2B0), fontSize: 11, fontWeight: FontWeight.bold))),
    ]),
  );
}
