plugins {
    id("com.android.library")
    id("org.jetbrains.kotlin.android")
    id("com.google.dagger.hilt.android")
    id("com.google.devtools.ksp")
}

android {
    namespace = "org.localforge.alicia.service.hotkey"
    compileSdk = 36
    buildToolsVersion = "36.0.0"

    defaultConfig {
        minSdk = 35

        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"
        consumerProguardFiles("consumer-rules.pro")
    }

    testOptions {
        targetSdk = 36
    }

    lint {
        targetSdk = 36
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
}

dependencies {
    // Module dependencies
    implementation(project(":core:common"))
    implementation(project(":core:domain"))
    implementation(project(":core:data"))

    // Core Android
    implementation("androidx.core:core-ktx:1.17.0")
    implementation("androidx.cardview:cardview:1.0.0")

    // Logging
    implementation("com.jakewharton.timber:timber:5.0.1")

    // Kotlin Coroutines
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-android:1.9.0")

    // DataStore
    implementation("androidx.datastore:datastore-preferences:1.2.0")

    // Dependency Injection - Hilt
    implementation("com.google.dagger:hilt-android:2.54")
    ksp("com.google.dagger:hilt-android-compiler:2.54")

    // Testing
    testImplementation("junit:junit:4.13.2")
    testImplementation("org.jetbrains.kotlinx:kotlinx-coroutines-test:1.9.0")
    androidTestImplementation("androidx.test.ext:junit:1.3.0")
    androidTestImplementation("androidx.test.espresso:espresso-core:3.7.0")
}
