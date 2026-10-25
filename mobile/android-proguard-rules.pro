# R8/ProGuard: Flutter/Dart AOT compilation already removes much Dart symbol metadata.
# Keep only rules required by native Java/Kotlin plugins; avoid blanket keep rules.
# Add plugin-specific keep rules only if release testing demonstrates a reflection issue.
-keepattributes *Annotation*,Signature,InnerClasses,EnclosingMethod
-dontwarn javax.annotation.**
