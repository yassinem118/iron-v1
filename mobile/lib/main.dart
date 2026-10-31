import 'package:flutter/material.dart';
import 'screens/splash_screen.dart';

const ironBackground = Color(0xFF080D16);
const ironPanel = Color(0xFF111A29);
const ironAccent = Color(0xFFFFD400);

void main() => runApp(const IronApp());

class IronApp extends StatelessWidget {
  const IronApp({super.key});

  @override
  Widget build(BuildContext context) => MaterialApp(
        title: 'IRON',
        debugShowCheckedModeBanner: false,
        theme: ThemeData(
          useMaterial3: true,
          brightness: Brightness.dark,
          scaffoldBackgroundColor: ironBackground,
          colorScheme: ColorScheme.fromSeed(
            seedColor: ironAccent,
            brightness: Brightness.dark,
            surface: ironPanel,
          ),
          appBarTheme: const AppBarTheme(
            backgroundColor: ironBackground,
            foregroundColor: Colors.white,
            centerTitle: false,
          ),
        ),
        home: const SplashScreen(),
      );
}
