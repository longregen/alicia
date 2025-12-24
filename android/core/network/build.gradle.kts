plugins {
    id("com.android.library")
    id("org.jetbrains.kotlin.android")
    id("com.google.dagger.hilt.android")
    id("com.google.devtools.ksp")
}

android {
    namespace = "org.localforge.alicia.core.network"
    compileSdk = 36
    buildToolsVersion = "36.0.0"

    defaultConfig {
        minSdk = 35

        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"
        consumerProguardFiles("consumer-rules.pro")

        buildConfigField("String", "API_BASE_URL", "\"https://api.example.com\"")
        buildConfigField("String", "LIVEKIT_URL", "\"wss://livekit.example.com\"")
    }

    buildTypes {
        release {
            isMinifyEnabled = false
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro"
            )
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
        buildConfig = true
    }
}

dependencies {
    // Module dependencies
    implementation(project(":core:common"))
    implementation(project(":core:domain"))

    // Core Android
    implementation("androidx.core:core-ktx:1.17.0")

    // Kotlin Coroutines
    api("org.jetbrains.kotlinx:kotlinx-coroutines-android:1.9.0")

    // LiveKit
    api("io.livekit:livekit-android:2.9.0")

    // MessagePack
    api("org.msgpack:msgpack-core:0.9.6")
    api("com.ensarsarajcic.kotlinx:serialization-msgpack:0.5.3")
    api("org.jetbrains.kotlinx:kotlinx-serialization-json:1.8.1")

    // Networking - Retrofit
    api("com.squareup.retrofit2:retrofit:2.9.0")
    api("com.squareup.retrofit2:converter-moshi:2.9.0")
    api("com.squareup.okhttp3:okhttp:4.12.0")
    api("com.squareup.okhttp3:logging-interceptor:4.12.0")

    // JSON - Moshi
    api("com.squareup.moshi:moshi:1.15.0")
    ksp("com.squareup.moshi:moshi-kotlin-codegen:1.15.0")

    // Logging
    api("com.jakewharton.timber:timber:5.0.1")

    // Dependency Injection - Hilt
    implementation("com.google.dagger:hilt-android:2.54")
    ksp("com.google.dagger:hilt-android-compiler:2.54")

    // Testing
    testImplementation("junit:junit:4.13.2")
    testImplementation("org.jetbrains.kotlinx:kotlinx-coroutines-test:1.9.0")
    testImplementation("com.squareup.okhttp3:mockwebserver:4.12.0")
    testImplementation("io.mockk:mockk:1.13.8")
    androidTestImplementation("androidx.test.ext:junit:1.3.0")
    androidTestImplementation("androidx.test.espresso:espresso-core:3.7.0")
}

// Workaround for Moshi KAPT deprecation warning triggered by Hilt's hiltJavaCompile task
// Hilt incorrectly loads KSP dependencies and passes them to JavaCompile, causing the warning
// See: https://github.com/square/moshi/issues/1779
configurations.configureEach {
    if (name.startsWith("kapt") || name.contains("AnnotationProcessor")) {
        exclude(group = "com.squareup.moshi", module = "moshi-kotlin-codegen")
    }
}
