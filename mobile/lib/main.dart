import 'package:flutter/material.dart';
import 'screens/splash_screen.dart';

void main() => runApp(const IronApp());
class IronApp extends StatelessWidget {
  const IronApp({super.key});
  @override Widget build(BuildContext context) => MaterialApp(
    title: 'Iron v1', debugShowCheckedModeBanner: false,
    theme: ThemeData(colorScheme: ColorScheme.fromSeed(seedColor: const Color(0xFFFF4D32), brightness: Brightness.dark), scaffoldBackgroundColor: const Color(0xFF080C12), useMaterial3: true),
    home: const SplashScreen(),
  );
}
