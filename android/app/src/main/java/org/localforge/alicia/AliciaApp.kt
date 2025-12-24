package org.localforge.alicia

import android.app.Application
import dagger.hilt.android.HiltAndroidApp
import timber.log.Timber

/**
 * Alicia Android Application
 * 
 * Main application class configured with Hilt dependency injection.
 * This is the entry point for the Android application.
 */
@HiltAndroidApp
class AliciaApp : Application() {

    override fun onCreate() {
        super.onCreate()
        
        // Initialize logging in debug builds
        if (BuildConfig.DEBUG) {
            Timber.plant(Timber.DebugTree())
        }
        
        Timber.d("Alicia app initialized")
    }
}
