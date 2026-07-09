import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:guard_app/screens/widgets/scenario_card.dart';

void main() {
  testWidgets('点击ScenarioCard后立即显示"已触发"反馈，1.2秒后恢复', (tester) async {
    bool tapped = false;

    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: ScenarioCard(
            icon: Icons.warning,
            iconColor: Colors.red,
            title: '测试场景',
            subtitle: '测试说明',
            onTap: () => tapped = true,
          ),
        ),
      ),
    );

    expect(find.text('已触发'), findsNothing);
    expect(find.byIcon(Icons.chevron_right_rounded), findsOneWidget);

    await tester.tap(find.byType(ScenarioCard));
    await tester.pump();

    expect(tapped, isTrue, reason: 'onTap回调应该被调用');
    expect(find.text('已触发'), findsOneWidget, reason: '点击后应该立即显示"已触发"反馈文字');
    expect(find.byIcon(Icons.check_circle_rounded), findsOneWidget);

    await tester.pump(const Duration(milliseconds: 1300));
    expect(find.text('已触发'), findsNothing, reason: '1.2秒后反馈应该消失，恢复默认箭头图标');
    expect(find.byIcon(Icons.chevron_right_rounded), findsOneWidget);
  });
}
