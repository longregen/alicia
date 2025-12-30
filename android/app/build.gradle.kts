plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.android")
    id("org.jetbrains.kotlin.plugin.compose")
    id("com.google.dagger.hilt.android")
    id("com.google.devtools.ksp")
}

android {
    namespace = "org.localforge.alicia"
    compileSdk = 36
    buildToolsVersion = "36.0.0"

    defaultConfig {
        applicationId = "org.localforge.alicia"
        minSdk = 35
        targetSdk = 36
        versionCode = 1
        versionName = "1.0.0"

        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"
        vectorDrawables {
            useSupportLibrary = true
        }

        // BuildConfig fields
        buildConfigField("String", "LIVEKIT_URL", "\"wss://livekit.example.com\"")
        buildConfigField("String", "API_BASE_URL", "\"https://api.example.com\"")
        buildConfigField("String", "PORCUPINE_ACCESS_KEY", "\"\"")

        // Limit resources to English only for smaller APK
        androidResources {
            localeFilters += listOf("en")
        }
    }

    // Generate separate APKs per ABI for smaller download sizes
    splits {
        abi {
            isEnable = true
            reset()
            include("armeabi-v7a", "arm64-v8a", "x86_64")
            isUniversalApk = true
        }
    }

    buildTypes {
        release {
            isMinifyEnabled = true
            isShrinkResources = true
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro"
            )
        }
        debug {
            isMinifyEnabled = false
            applicationIdSuffix = ".debug"
            versionNameSuffix = "-debug"
        }
        create("minifiedDebug") {
            // Minified but debuggable - best of both worlds
            initWith(getByName("debug"))
            isMinifyEnabled = true
            isShrinkResources = true
            isDebuggable = true
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro"
            )
            matchingFallbacks += listOf("debug")
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    kotlinOptions {
        jvmTarget = "17"
    }

    buildFeatures {
        compose = true
        buildConfig = true
    }

    lint {
        abortOnError = false
        checkDependencies = false  // Reduce memory usage by not checking all dependencies
        warningsAsErrors = false
        // Disable heavy reports to reduce memory usage
        htmlReport = false
        xmlReport = false
        textReport = true
    }

    packaging {
        resources {
            excludes += "/META-INF/{AL2.0,LGPL2.1}"
            excludes += "/META-INF/DEPENDENCIES"
            excludes += "/META-INF/LICENSE*"
            excludes += "/META-INF/NOTICE*"
            excludes += "/META-INF/*.kotlin_module"
        }
        jniLibs {
            // Don't attempt to strip native libraries that can't be stripped
            useLegacyPackaging = false
        }
    }
}

dependencies {
    // Core Android
    implementation("androidx.core:core-ktx:1.17.0")
    implementation("androidx.lifecycle:lifecycle-runtime-ktx:2.10.0")
    implementation("androidx.activity:activity-ktx:1.12.2")

    // Kotlin Coroutines
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-android:1.9.0")
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-core:1.9.0")

    // Compose BOM
    implementation(platform("androidx.compose:compose-bom:2025.12.01"))
    implementation("androidx.compose.ui:ui")
    implementation("androidx.compose.ui:ui-graphics")
    implementation("androidx.compose.ui:ui-tooling-preview")
    implementation("androidx.compose.material3:material3")
    implementation("androidx.compose.material:material-icons-core")
    implementation("androidx.activity:activity-compose:1.12.2")
    implementation("androidx.lifecycle:lifecycle-viewmodel-compose:2.10.0")
    implementation("androidx.lifecycle:lifecycle-runtime-compose:2.10.0")
    implementation("androidx.navigation:navigation-compose:2.9.6")

    // LiveKit
    implementation("io.livekit:livekit-android:2.9.0")

    // Wake word detection - Porcupine
    implementation("ai.picovoice:porcupine-android:3.0.0")

    // MessagePack
    implementation("org.msgpack:msgpack-core:0.9.6")
    implementation("com.ensarsarajcic.kotlinx:serialization-msgpack:0.5.3")
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-json:1.8.1")

    // Networking - Retrofit
    implementation("com.squareup.retrofit2:retrofit:2.9.0")
    implementation("com.squareup.retrofit2:converter-moshi:2.9.0")
    implementation("com.squareup.okhttp3:okhttp:4.12.0")
    implementation("com.squareup.okhttp3:logging-interceptor:4.12.0")

    // JSON - Moshi
    implementation("com.squareup.moshi:moshi:1.15.0")
    ksp("com.squareup.moshi:moshi-kotlin-codegen:1.15.0")

    // Database - Room
    implementation("androidx.room:room-runtime:2.8.4")
    implementation("androidx.room:room-ktx:2.8.4")
    ksp("androidx.room:room-compiler:2.8.4")

    // Dependency Injection - Hilt
    implementation("com.google.dagger:hilt-android:2.54")
    ksp("com.google.dagger:hilt-android-compiler:2.54")
    implementation("androidx.hilt:hilt-navigation-compose:1.3.0")
    implementation("androidx.hilt:hilt-work:1.3.0")
    ksp("androidx.hilt:hilt-compiler:1.3.0")

    // DataStore (Preferences)
    implementation("androidx.datastore:datastore-preferences:1.2.0")

    // WorkManager
    implementation("androidx.work:work-runtime-ktx:2.11.0")

    // Accessibility
    implementation("androidx.core:core-ktx:1.17.0")

    // Accompanist (for permissions)
    implementation("com.google.accompanist:accompanist-permissions:0.34.0")

    // Logging
    implementation("com.jakewharton.timber:timber:5.0.1")

    // Module dependencies
    implementation(project(":core:common"))
    implementation(project(":core:data"))
    implementation(project(":core:domain"))
    implementation(project(":core:network"))
    implementation(project(":core:database"))
    implementation(project(":feature:assistant"))
    implementation(project(":feature:conversations"))
    implementation(project(":feature:settings"))
    implementation(project(":service:voice"))
    implementation(project(":service:hotkey"))

    // Testing
    testImplementation("junit:junit:4.13.2")
    testImplementation("org.jetbrains.kotlinx:kotlinx-coroutines-test:1.9.0")
    testImplementation("com.google.truth:truth:1.1.5")
    testImplementation("io.mockk:mockk:1.13.8")

    androidTestImplementation("androidx.test.ext:junit:1.3.0")
    androidTestImplementation("androidx.test.espresso:espresso-core:3.7.0")
    androidTestImplementation("androidx.test.uiautomator:uiautomator:2.3.0")
    androidTestImplementation("androidx.test:runner:1.7.0")
    androidTestImplementation("androidx.test:rules:1.7.0")
    androidTestImplementation(platform("androidx.compose:compose-bom:2025.12.01"))
    androidTestImplementation("androidx.compose.ui:ui-test-junit4")

    debugImplementation("androidx.compose.ui:ui-tooling")
    debugImplementation("androidx.compose.ui:ui-test-manifest")
}

// Workaround for Moshi KAPT deprecation warning triggered by Hilt's hiltJavaCompile task
// Hilt incorrectly loads KSP dependencies and passes them to JavaCompile, causing the warning
// See: https://github.com/square/moshi/issues/1779
configurations.configureEach {
    if (name.startsWith("kapt") || name.contains("AnnotationProcessor")) {
        exclude(group = "com.squareup.moshi", module = "moshi-kotlin-codegen")
    }
}
