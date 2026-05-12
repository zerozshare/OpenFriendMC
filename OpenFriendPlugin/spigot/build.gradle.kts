plugins {
    java
}

repositories {
    maven("https://hub.spigotmc.org/nexus/content/repositories/snapshots/")
    maven("https://oss.sonatype.org/content/repositories/snapshots/")
}

val pluginVersion: String = (project.findProperty("plugin.version") as String?) ?: "0.1.0"
val mcVersion: String = (project.findProperty("mc.version") as String?) ?: "1.8.8"
val javaRelease: String = (project.findProperty("java.release") as String?) ?: "8"
val spigotApi: String = (project.findProperty("spigot.api") as String?) ?: "1.8.8-R0.1-SNAPSHOT"

version = pluginVersion

dependencies {
    implementation(project(":commons"))
    compileOnly("org.spigotmc:spigot-api:$spigotApi") {
        isTransitive = false
    }
}

extensions.configure<JavaPluginExtension> {
    toolchain.languageVersion.set(JavaLanguageVersion.of(21))
}

tasks.withType<JavaCompile>().configureEach {
    options.encoding = "UTF-8"
    options.release.set(javaRelease.toInt())
}

tasks.processResources {
    val mapping = mapOf(
        "pluginVersion" to pluginVersion,
        "mcVersion" to mcVersion,
    )
    inputs.properties(mapping)
    filesMatching("plugin.yml") {
        expand(mapping)
    }
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
    archiveBaseName.set("OpenFriend-spigot-${mcVersion}")
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
