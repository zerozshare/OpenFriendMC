plugins {
    java
}

repositories {
    maven("https://repo.papermc.io/repository/maven-public/")
}

val pluginVersion: String = (project.findProperty("plugin.version") as String?) ?: "0.1.0"
val velocityApi: String = (project.findProperty("velocity.api") as String?) ?: "3.3.0-SNAPSHOT"

version = pluginVersion

dependencies {
    implementation(project(":commons"))
    compileOnly("com.velocitypowered:velocity-api:$velocityApi")
    annotationProcessor("com.velocitypowered:velocity-api:$velocityApi")
}

extensions.configure<JavaPluginExtension> {
    toolchain.languageVersion.set(JavaLanguageVersion.of(21))
}

tasks.withType<JavaCompile>().configureEach {
    options.encoding = "UTF-8"
    options.release.set(17)
}

tasks.register<Copy>("copyBinaries") {
    description = "Stage Go binaries from bin/dist/ into resources/openfriend/bin/."
    val distDir = rootProject.file("../bin/dist")
    from(distDir) {
        include("openfriend-*")
    }
    into(layout.projectDirectory.dir("src/main/resources/openfriend/bin"))
}

tasks.processResources {
    dependsOn("copyBinaries")
}

tasks.jar {
    archiveBaseName.set("OpenFriend-velocity-${pluginVersion}")
    archiveVersion.set("")
    archiveClassifier.set("")
    dependsOn(configurations.runtimeClasspath)
    from({
        configurations.runtimeClasspath.get().filter {
            it.isFile
        }.map { zipTree(it) }
    })
    duplicatesStrategy = DuplicatesStrategy.EXCLUDE
}
