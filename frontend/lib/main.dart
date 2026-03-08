import 'package:flutter/material.dart';

void main() {
  runApp(const ZeclawApp());
}

class ZeclawApp extends StatelessWidget {
  const ZeclawApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Zeclaw',
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: Colors.deepPurple),
        useMaterial3: true,
      ),
      home: const Scaffold(
        body: Center(
          child: Text('Zeclaw'),
        ),
      ),
    );
  }
}
