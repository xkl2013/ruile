package com.ruile.ruile_mobile

import io.flutter.embedding.engine.FlutterEngine
import io.flutter.embedding.android.FlutterActivity

class MainActivity : FlutterActivity() {
    private var jieliRecordingCardSdkBridge: JieliRecordingCardSdkBridge? = null

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        jieliRecordingCardSdkBridge = JieliRecordingCardSdkBridge(this).also {
            it.register(flutterEngine)
        }
    }

    override fun onDestroy() {
        jieliRecordingCardSdkBridge?.release()
        jieliRecordingCardSdkBridge = null
        super.onDestroy()
    }
}
