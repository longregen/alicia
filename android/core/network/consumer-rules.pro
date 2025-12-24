# Consumer proguard rules for network module

# Retrofit
-keepattributes Signature, InnerClasses, EnclosingMethod
-keepattributes RuntimeVisibleAnnotations, RuntimeVisibleParameterAnnotations

# Moshi
-keep class org.localforge.alicia.core.network.model.** { *; }
-keepclassmembers class org.localforge.alicia.core.network.model.** {
    <init>(...);
    <fields>;
}

# Protocol models
-keep class org.localforge.alicia.core.network.protocol.** { *; }
-keep class org.localforge.alicia.core.network.protocol.bodies.** { *; }

# API Service
-keep interface org.localforge.alicia.core.network.api.AliciaApiService { *; }
