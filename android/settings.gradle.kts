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
        maven { url = uri("https://jitpack.io") }
    }
}

rootProject.name = "Alicia"

include(":app")

// Core modules
include(":core:common")
include(":core:data")
include(":core:domain")
include(":core:network")
include(":core:database")

// Feature modules
include(":feature:assistant")
include(":feature:conversations")
include(":feature:settings")

// Service modules
include(":service:voice")
include(":service:hotkey")
