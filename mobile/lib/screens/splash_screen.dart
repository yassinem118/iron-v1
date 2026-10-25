import 'package:flutter/material.dart';
import 'login_screen.dart';

class SplashScreen extends StatefulWidget { const SplashScreen({super.key}); @override State<SplashScreen> createState()=>_SplashScreenState(); }
class _SplashScreenState extends State<SplashScreen> with SingleTickerProviderStateMixin {
  late final AnimationController _controller = AnimationController(vsync:this,duration:const Duration(milliseconds:900))..forward();
  @override void initState(){super.initState();Future.delayed(const Duration(milliseconds:1200),(){if(mounted)Navigator.of(context).pushReplacement(MaterialPageRoute(builder:(_)=>const LoginScreen()));});}
  @override void dispose(){_controller.dispose();super.dispose();}
  @override Widget build(BuildContext context)=>Scaffold(body:Center(child:FadeTransition(opacity:_controller,child:ScaleTransition(scale:Tween<double>(begin:.8,end:1).animate(CurvedAnimation(parent:_controller,curve:Curves.easeOutBack)),child:Column(mainAxisSize:MainAxisSize.min,children:[Container(width:88,height:88,decoration:BoxDecoration(color:const Color(0xFFFF4D32),borderRadius:BorderRadius.circular(24)),child:const Icon(Icons.shield_moon,size:48,color:Colors.white)),const SizedBox(height:20),const Text('IRON v1',style:TextStyle(fontSize:28,fontWeight:FontWeight.w900,letterSpacing:4)),const SizedBox(height:8),const Text('SECURE ACCESS SYSTEM',style:TextStyle(color:Colors.white54,fontSize:10,letterSpacing:2))])))));
}
