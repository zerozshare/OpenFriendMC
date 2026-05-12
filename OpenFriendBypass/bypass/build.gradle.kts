plugins {
    java
}

repositories {
    maven("https://hub.spigotmc.org/nexus/content/repositories/snapshots/")
    maven("https://oss.sonatype.org/content/repositories/snapshots/")
    maven("https://repo.codemc.io/repository/maven-releases/")
    maven("https://repo.papermc.io/repository/maven-public/")
}

val pluginVersion: String = (project.findProperty("plugin.version") as String?) ?: "0.1.0"
val mcVersion: String = (project.findProperty("mc.version") as String?) ?: "1.21.4"
val javaRelease: String = (project.findProperty("java.release") as String?) ?: "21"
val spigotApi: String = (project.findProperty("spigot.api") as String?) ?: "1.21.4-R0.1-SNAPSHOT"
val packetEventsVersion = "2.7.0"

version = pluginVersion

dependencies {
    implementation(project(":commons"))
    compileOnly("org.spigotmc:spigot-api:$spigotApi") {
        isTransitive = false
    }
    compileOnly("com.github.retrooper:packetevents-spigot:$packetEventsVersion")
    compileOnly("io.netty:netty-all:4.1.107.Final")
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

tasks.jar {
    archiveBaseName.set("OpenFriendBypass-${mcVersion}")
    archiveVersion.set("")
    archiveClassifier.set("")
    dependsOn(configurations.runtimeClasspath)
    from({
        configurations.runtimeClasspath.get().filter { it.isFile }.map { zipTree(it) }
    })
    duplicatesStrategy = DuplicatesStrategy.EXCLUDE
}
