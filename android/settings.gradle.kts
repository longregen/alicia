pluginManagement {
    repositories {
        google()
        mavenCentral()
        gradlePluginPortal()
    }
}

dependencyResolutionManagement {
    repositoriesMode.set(RepositoriesMode.FAIL_ON_PROJECT_REPOS)
    repositories {
        google()
        mavenCentral()
        // Vosk Speech Recognition repository
        maven { url = uri("https://alphacephei.com/maven/") }
    }
}

rootProject.name = "Alicia"
include(":app")
