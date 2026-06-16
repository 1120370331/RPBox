import fs from 'node:fs'
import path from 'node:path'

const platform = (process.argv[2] || 'all').toLowerCase()
const cwd = process.cwd()
const mobileRoot = path.basename(cwd) === 'mobile' ? cwd : path.join(cwd, 'mobile')
const appId = 'app.rpbox.mobile'
const appPackage = appId
const associatedHosts = ['totalrpbox.com', 'www.totalrpbox.com']
const appLinkPathPrefixes = ['/posts/', '/items/', '/stories/', '/profiles/', '/guild/', '/open-app.html']

function ensureFile(filePath, contents) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true })
  fs.writeFileSync(filePath, contents, 'utf8')
}

function buildAndroidAppLinkBlock() {
  const filters = [
    `            <intent-filter>
                <action android:name="android.intent.action.VIEW" />
                <category android:name="android.intent.category.DEFAULT" />
                <category android:name="android.intent.category.BROWSABLE" />
                <data android:scheme="${appId}" />
            </intent-filter>`,
  ]

  for (const host of associatedHosts) {
    for (const pathPrefix of appLinkPathPrefixes) {
      filters.push(`            <intent-filter android:autoVerify="true">
                <action android:name="android.intent.action.VIEW" />
                <category android:name="android.intent.category.DEFAULT" />
                <category android:name="android.intent.category.BROWSABLE" />
                <data android:scheme="https" android:host="${host}" android:pathPrefix="${pathPrefix}" />
            </intent-filter>`)
    }
  }

  return ['            <!-- RPBOX_APP_LINKS_START -->', ...filters, '            <!-- RPBOX_APP_LINKS_END -->'].join('\n')
}

function ensureAndroidManifestPermission(manifest, permissionName) {
  if (manifest.includes(`android:name="${permissionName}"`)) {
    return manifest
  }
  return manifest.replace(
    /<application\b/,
    `    <uses-permission android:name="${permissionName}" />\n\n    <application`,
  )
}

function buildAndroidFileProviderBlock() {
  return `        <provider
            android:name="androidx.core.content.FileProvider"
            android:authorities="\${applicationId}.fileprovider"
            android:exported="false"
            android:grantUriPermissions="true">
            <meta-data
                android:name="android.support.FILE_PROVIDER_PATHS"
                android:resource="@xml/file_paths"></meta-data>
        </provider>`
}

function ensureAndroidFileProviderBlock(manifest) {
  const providerBlock = buildAndroidFileProviderBlock()
  let replaced = false

  manifest = manifest.replace(/<provider\b[\s\S]*?<\/provider>/g, (block) => {
    if (replaced || !block.includes('androidx.core.content.FileProvider')) {
      return block
    }
    replaced = true
    return providerBlock
  })

  if (replaced) {
    return manifest
  }

  return manifest.replace(/<\/application>/, `${providerBlock}\n    </application>`)
}

function removeAndroidUpdaterProviderBlock(manifest) {
  return manifest.replace(/\s*<!-- RPBOX_UPDATER_PROVIDER_START -->[\s\S]*?<!-- RPBOX_UPDATER_PROVIDER_END -->/g, '')
}

function buildAndroidUpdaterPluginSource() {
  return `package ${appPackage};

import android.content.Context;
import android.content.Intent;
import android.net.Uri;
import android.os.Build;
import android.provider.Settings;

import androidx.core.content.FileProvider;

import com.getcapacitor.JSObject;
import com.getcapacitor.Plugin;
import com.getcapacitor.PluginCall;
import com.getcapacitor.PluginMethod;
import com.getcapacitor.annotation.CapacitorPlugin;

import java.io.File;
import java.io.FileOutputStream;
import java.io.InputStream;
import java.net.HttpURLConnection;
import java.net.URL;
import java.util.Locale;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

@CapacitorPlugin(name = "RPBoxUpdater")
public class RPBoxUpdaterPlugin extends Plugin {
    private final ExecutorService executor = Executors.newSingleThreadExecutor();

    @PluginMethod
    public void getInstallPermissionStatus(PluginCall call) {
        JSObject result = new JSObject();
        boolean required = Build.VERSION.SDK_INT >= Build.VERSION_CODES.O;
        result.put("required", required);
        result.put("granted", !required || getContext().getPackageManager().canRequestPackageInstalls());
        call.resolve(result);
    }

    @PluginMethod
    public void requestInstallPermission(PluginCall call) {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) {
            JSObject result = new JSObject();
            result.put("opened", false);
            result.put("granted", true);
            call.resolve(result);
            return;
        }

        Intent intent = new Intent(
            Settings.ACTION_MANAGE_UNKNOWN_APP_SOURCES,
            Uri.parse("package:" + getContext().getPackageName())
        );
        intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK);
        getContext().startActivity(intent);

        JSObject result = new JSObject();
        result.put("opened", true);
        result.put("granted", getContext().getPackageManager().canRequestPackageInstalls());
        call.resolve(result);
    }

    @PluginMethod
    public void downloadAndInstall(PluginCall call) {
        String rawUrl = call.getString("url", "");
        String version = call.getString("version", "latest");
        if (rawUrl.trim().isEmpty()) {
            call.reject("Missing update URL");
            return;
        }
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O && !getContext().getPackageManager().canRequestPackageInstalls()) {
            call.reject("INSTALL_PERMISSION_REQUIRED", "INSTALL_PERMISSION_REQUIRED");
            return;
        }

        executor.execute(() -> {
            HttpURLConnection connection = null;
            File apkFile = null;
            try {
                URL url = new URL(rawUrl);
                connection = (HttpURLConnection) url.openConnection();
                connection.setConnectTimeout(15000);
                connection.setReadTimeout(30000);
                connection.setInstanceFollowRedirects(true);
                connection.connect();

                int status = connection.getResponseCode();
                if (status < 200 || status >= 300) {
                    throw new IllegalStateException("HTTP " + status);
                }

                long totalBytes = connection.getContentLengthLong();
                File updateDir = new File(resolveUpdateDirectory(), "rpbox-updates");
                if (!updateDir.exists() && !updateDir.mkdirs()) {
                    throw new IllegalStateException("Failed to create update directory");
                }
                apkFile = new File(updateDir, buildApkFileName(version));

                try (
                    InputStream input = connection.getInputStream();
                    FileOutputStream output = new FileOutputStream(apkFile, false)
                ) {
                    byte[] buffer = new byte[64 * 1024];
                    long downloadedBytes = 0;
                    int read;
                    long lastEmitAt = 0;
                    while ((read = input.read(buffer)) != -1) {
                        output.write(buffer, 0, read);
                        downloadedBytes += read;
                        long now = System.currentTimeMillis();
                        if (now - lastEmitAt > 250 || downloadedBytes == totalBytes) {
                            emitProgress(downloadedBytes, totalBytes, false);
                            lastEmitAt = now;
                        }
                    }
                    output.flush();
                    emitProgress(downloadedBytes, totalBytes, true);
                }

                installApk(apkFile);
                JSObject result = new JSObject();
                result.put("path", apkFile.getAbsolutePath());
                call.resolve(result);
            } catch (Exception error) {
                if (apkFile != null && apkFile.exists()) {
                    //noinspection ResultOfMethodCallIgnored
                    apkFile.delete();
                }
                call.reject(error.getMessage(), error);
            } finally {
                if (connection != null) {
                    connection.disconnect();
                }
            }
        });
    }

    private File resolveUpdateDirectory() {
        Context context = getContext();
        File dir = context.getExternalCacheDir();
        return dir != null ? dir : context.getCacheDir();
    }

    private String buildApkFileName(String version) {
        String safeVersion = version == null ? "latest" : version.replaceAll("[^A-Za-z0-9._-]", "_");
        if (safeVersion.trim().isEmpty()) {
            safeVersion = "latest";
        }
        return String.format(Locale.US, "RPBox_%s_android.apk", safeVersion);
    }

    private void emitProgress(long downloadedBytes, long totalBytes, boolean finished) {
        JSObject data = new JSObject();
        data.put("downloadedBytes", downloadedBytes);
        data.put("totalBytes", totalBytes);
        data.put("finished", finished);
        if (totalBytes > 0) {
            data.put("percent", Math.min(100.0, (downloadedBytes * 100.0) / totalBytes));
        } else {
            data.put("percent", 0);
        }
        notifyListeners("downloadProgress", data);
    }

    private void installApk(File apkFile) {
        Context context = getContext();
        Uri apkUri = FileProvider.getUriForFile(
            context,
            context.getPackageName() + ".fileprovider",
            apkFile
        );
        Intent intent = new Intent(Intent.ACTION_VIEW);
        intent.setDataAndType(apkUri, "application/vnd.android.package-archive");
        intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK);
        intent.addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION);
        context.startActivity(intent);
    }
}
`
}

function patchAndroidMainActivity() {
  const candidatePaths = [
    path.join(mobileRoot, 'android', 'app', 'src', 'main', 'java', ...appPackage.split('.'), 'MainActivity.java'),
    path.join(mobileRoot, 'android', 'app', 'src', 'main', 'java', 'com', 'getcapacitor', 'myapp', 'MainActivity.java'),
  ]
  let mainActivityPath = candidatePaths.find((candidate) => fs.existsSync(candidate))
  if (!mainActivityPath) {
    const rootDir = path.join(mobileRoot, 'android', 'app', 'src', 'main', 'java')
    const stack = fs.existsSync(rootDir) ? [rootDir] : []
    while (stack.length > 0 && !mainActivityPath) {
      const current = stack.pop()
      if (!current) break
      for (const entry of fs.readdirSync(current, { withFileTypes: true })) {
        const entryPath = path.join(current, entry.name)
        if (entry.isDirectory()) {
          stack.push(entryPath)
        } else if (entry.isFile() && entry.name === 'MainActivity.java') {
          mainActivityPath = entryPath
          break
        }
      }
    }
  }

  if (!mainActivityPath || !fs.existsSync(mainActivityPath)) return

  let source = fs.readFileSync(mainActivityPath, 'utf8')
  const packageMatch = source.match(/^package\s+([A-Za-z0-9_.]+);/m)
  const sourcePackage = packageMatch ? packageMatch[1] : ''
  const needsPluginImport = sourcePackage !== appPackage

  if (needsPluginImport && !source.includes(`import ${appPackage}.RPBoxUpdaterPlugin;`)) {
    if (source.includes('import com.getcapacitor.BridgeActivity;')) {
      source = source.replace(
        /import com\.getcapacitor\.BridgeActivity;\s*/,
        `import com.getcapacitor.BridgeActivity;\nimport ${appPackage}.RPBoxUpdaterPlugin;\n`,
      )
    } else {
      source = source.replace(
        /^package\s+[A-Za-z0-9_.]+;\s*/m,
        `$&\nimport ${appPackage}.RPBoxUpdaterPlugin;\n`,
      )
    }
  }

  if (!source.includes('import android.os.Bundle;')) {
    if (source.includes('import com.getcapacitor.BridgeActivity;')) {
      source = source.replace(
        /import com\.getcapacitor\.BridgeActivity;\s*/,
        'import android.os.Bundle;\n\nimport com.getcapacitor.BridgeActivity;\n',
      )
    } else {
      source = source.replace(
        /^package\s+[A-Za-z0-9_.]+;\s*/m,
        `$&\nimport android.os.Bundle;\n`,
      )
    }
  }

  if (/registerPlugin\(RPBoxUpdaterPlugin\.class\)/.test(source)) {
    fs.writeFileSync(mainActivityPath, source, 'utf8')
    return
  }

  if (/void\s+onCreate\s*\(\s*Bundle\s+savedInstanceState\s*\)\s*\{/.test(source)) {
    source = source.replace(
      /(void\s+onCreate\s*\(\s*Bundle\s+savedInstanceState\s*\)\s*\{)/,
      `$1
        registerPlugin(RPBoxUpdaterPlugin.class);`,
    )
  } else if (/public class MainActivity extends BridgeActivity\s*\{\s*\}/.test(source)) {
    source = source.replace(
      /public class MainActivity extends BridgeActivity\s*\{\s*\}/,
      `public class MainActivity extends BridgeActivity {
    @Override
    public void onCreate(Bundle savedInstanceState) {
        registerPlugin(RPBoxUpdaterPlugin.class);
        super.onCreate(savedInstanceState);
    }
}`,
    )
  } else if (/public class MainActivity extends BridgeActivity\s*\{/.test(source)) {
    source = source.replace(
      /public class MainActivity extends BridgeActivity\s*\{/,
      `public class MainActivity extends BridgeActivity {
    @Override
    public void onCreate(Bundle savedInstanceState) {
        registerPlugin(RPBoxUpdaterPlugin.class);
        super.onCreate(savedInstanceState);
    }
`,
    )
  }

  if (!/registerPlugin\(RPBoxUpdaterPlugin\.class\)/.test(source)) {
    throw new Error(`Failed to inject RPBoxUpdaterPlugin registration into ${mainActivityPath}`)
  }

  fs.writeFileSync(mainActivityPath, source, 'utf8')
}

function patchAndroidUpdaterPlugin() {
  const javaDir = path.join(mobileRoot, 'android', 'app', 'src', 'main', 'java', ...appPackage.split('.'))
  ensureFile(path.join(javaDir, 'RPBoxUpdaterPlugin.java'), buildAndroidUpdaterPluginSource())
  patchAndroidFileProviderPaths()
  const legacyPathsPath = path.join(mobileRoot, 'android', 'app', 'src', 'main', 'res', 'xml', 'rpbox_updater_paths.xml')
  if (fs.existsSync(legacyPathsPath)) {
    fs.unlinkSync(legacyPathsPath)
  }
  patchAndroidMainActivity()
}

function upsertAndroidPath(xml, tagName, name, pathValue) {
  if (new RegExp(`<${tagName}\\b[^>]*(?:android:)?name="${name}"`).test(xml)) {
    return xml
  }
  return xml.replace(
    /<\/paths>/,
    `    <${tagName} name="${name}" path="${pathValue}" />\n</paths>`,
  )
}

function patchAndroidFileProviderPaths() {
  const pathsPath = path.join(mobileRoot, 'android', 'app', 'src', 'main', 'res', 'xml', 'file_paths.xml')
  if (!fs.existsSync(pathsPath)) {
    ensureFile(pathsPath, `<?xml version="1.0" encoding="utf-8"?>
<paths xmlns:android="http://schemas.android.com/apk/res/android">
    <external-path name="my_images" path="." />
    <cache-path name="my_cache_images" path="." />
    <cache-path name="rpbox_update_cache" path="." />
    <external-cache-path name="rpbox_update_external_cache" path="." />
</paths>`)
    return
  }

  let xml = fs.readFileSync(pathsPath, 'utf8')
  xml = upsertAndroidPath(xml, 'cache-path', 'rpbox_update_cache', '.')
  xml = upsertAndroidPath(xml, 'external-cache-path', 'rpbox_update_external_cache', '.')
  fs.writeFileSync(pathsPath, xml, 'utf8')
}

function upsertPlistArray(plist, key, values) {
  const block = `\t<key>${key}</key>\n\t<array>\n${values.map((value) => `\t\t<string>${value}</string>`).join('\n')}\n\t</array>`
  const pattern = new RegExp(`\\t<key>${key}<\\/key>\\s*\\t<array>[\\s\\S]*?\\t<\\/array>`)

  if (pattern.test(plist)) {
    return plist.replace(pattern, block)
  }

  return plist.replace(/<\/dict>\s*<\/plist>\s*$/, `${block}\n</dict>\n</plist>\n`)
}

function upsertPlistString(plist, key, value) {
  const block = `\t<key>${key}</key>\n\t<string>${value}</string>`
  const pattern = new RegExp(`\\t<key>${key}<\\/key>\\s*\\t<string>[\\s\\S]*?<\\/string>`)

  if (pattern.test(plist)) {
    return plist.replace(pattern, block)
  }

  return plist.replace(/<\/dict>\s*<\/plist>\s*$/, `${block}\n</dict>\n</plist>\n`)
}

function patchAndroid() {
  const stringsPath = path.join(mobileRoot, 'android', 'app', 'src', 'main', 'res', 'values', 'strings.xml')
  if (!fs.existsSync(stringsPath)) return

  let xml = fs.readFileSync(stringsPath, 'utf8')
  if (/<string name="custom_url_scheme">.*?<\/string>/.test(xml)) {
    xml = xml.replace(/<string name="custom_url_scheme">.*?<\/string>/, `<string name="custom_url_scheme">${appId}</string>`)
  } else {
    xml = xml.replace(/<\/resources>/, `    <string name="custom_url_scheme">${appId}</string>\n</resources>`)
  }
  fs.writeFileSync(stringsPath, xml, 'utf8')

  const manifestPath = path.join(mobileRoot, 'android', 'app', 'src', 'main', 'AndroidManifest.xml')
  if (!fs.existsSync(manifestPath)) return

  let manifest = fs.readFileSync(manifestPath, 'utf8')
  manifest = ensureAndroidManifestPermission(manifest, 'android.permission.REQUEST_INSTALL_PACKAGES')
  const appLinkBlock = buildAndroidAppLinkBlock()

  if (/<!-- RPBOX_APP_LINKS_START -->[\s\S]*<!-- RPBOX_APP_LINKS_END -->/.test(manifest)) {
    manifest = manifest.replace(/<!-- RPBOX_APP_LINKS_START -->[\s\S]*<!-- RPBOX_APP_LINKS_END -->/, appLinkBlock)
  } else if (/<\/activity>/.test(manifest)) {
    manifest = manifest.replace(/<\/activity>/, `${appLinkBlock}\n        </activity>`)
  }
  manifest = removeAndroidUpdaterProviderBlock(manifest)
  manifest = ensureAndroidFileProviderBlock(manifest)

  fs.writeFileSync(manifestPath, manifest, 'utf8')
  patchAndroidUpdaterPlugin()
}

function patchIos() {
  const infoPlistPath = path.join(mobileRoot, 'ios', 'App', 'App', 'Info.plist')
  if (!fs.existsSync(infoPlistPath)) {
    if (platform === 'ios') {
      throw new Error(`Missing iOS Info.plist: ${infoPlistPath}`)
    }
    return
  }

  let plist = fs.readFileSync(infoPlistPath, 'utf8')
  const urlTypesBlock = `
	<key>CFBundleURLTypes</key>
	<array>
		<dict>
			<key>CFBundleURLName</key>
			<string>${appId}</string>
			<key>CFBundleURLSchemes</key>
			<array>
				<string>${appId}</string>
			</array>
		</dict>
	</array>`

  if (/<key>CFBundleURLTypes<\/key>\s*<array>[\s\S]*?<\/array>/.test(plist)) {
    plist = plist.replace(/<key>CFBundleURLTypes<\/key>\s*<array>[\s\S]*?<\/array>/, urlTypesBlock.trim())
  } else {
    plist = plist.replace(/<\/dict>\s*<\/plist>\s*$/, `${urlTypesBlock}\n</dict>\n</plist>\n`)
  }
  plist = upsertPlistString(plist, 'NSCameraUsageDescription', 'RPBox 需要访问相机，以便拍摄并上传帖子、道具和评论图片。')
  plist = upsertPlistString(plist, 'NSPhotoLibraryUsageDescription', 'RPBox 需要访问照片，以便选择并上传帖子、道具和评论图片。')
  plist = upsertPlistString(plist, 'NSPhotoLibraryAddUsageDescription', 'RPBox 需要访问照片，以便保存和处理中转图片。')
  fs.writeFileSync(infoPlistPath, plist, 'utf8')

  for (const key of ['NSCameraUsageDescription', 'NSPhotoLibraryUsageDescription', 'NSPhotoLibraryAddUsageDescription']) {
    if (!plist.includes(`<key>${key}</key>`)) {
      throw new Error(`Failed to inject ${key} into ${infoPlistPath}`)
    }
  }

  const entitlementsPath = path.join(mobileRoot, 'ios', 'App', 'App', 'App.entitlements')
  const associatedDomains = associatedHosts.map((host) => `applinks:${host}`)
  if (fs.existsSync(entitlementsPath)) {
    let entitlements = fs.readFileSync(entitlementsPath, 'utf8')
    entitlements = upsertPlistArray(entitlements, 'com.apple.developer.associated-domains', associatedDomains)
    fs.writeFileSync(entitlementsPath, entitlements, 'utf8')
  } else {
    ensureFile(
      entitlementsPath,
      `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
\t<key>com.apple.developer.associated-domains</key>
\t<array>
${associatedDomains.map((domain) => `\t\t<string>${domain}</string>`).join('\n')}
\t</array>
</dict>
</plist>
`,
    )
  }

  const pbxprojPath = path.join(mobileRoot, 'ios', 'App', 'App.xcodeproj', 'project.pbxproj')
  if (!fs.existsSync(pbxprojPath)) {
    throw new Error(`Missing iOS Xcode project: ${pbxprojPath}`)
  }

  let pbxproj = fs.readFileSync(pbxprojPath, 'utf8')
  if (/CODE_SIGN_ENTITLEMENTS = [^;]+;/.test(pbxproj)) {
    pbxproj = pbxproj.replace(/CODE_SIGN_ENTITLEMENTS = [^;]+;/g, 'CODE_SIGN_ENTITLEMENTS = App/App.entitlements;')
  } else {
    pbxproj = pbxproj.replace(/INFOPLIST_FILE = App\/Info\.plist;/g, 'INFOPLIST_FILE = App/Info.plist;\n\t\t\t\tCODE_SIGN_ENTITLEMENTS = App/App.entitlements;')
  }
  fs.writeFileSync(pbxprojPath, pbxproj, 'utf8')

  if (!pbxproj.includes('CODE_SIGN_ENTITLEMENTS = App/App.entitlements;')) {
    throw new Error(`Failed to inject CODE_SIGN_ENTITLEMENTS into ${pbxprojPath}`)
  }

  const privacyManifestPath = path.join(mobileRoot, 'ios', 'App', 'PrivacyInfo.xcprivacy')
  if (!fs.existsSync(privacyManifestPath)) {
    ensureFile(
      privacyManifestPath,
      `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
  <dict>
    <key>NSPrivacyAccessedAPITypes</key>
    <array>
      <dict>
        <key>NSPrivacyAccessedAPIType</key>
        <string>NSPrivacyAccessedAPICategoryFileTimestamp</string>
        <key>NSPrivacyAccessedAPITypeReasons</key>
        <array>
          <string>C617.1</string>
        </array>
      </dict>
    </array>
  </dict>
</plist>
`,
    )
  }
}

if (platform === 'android' || platform === 'all') {
  patchAndroid()
}

if (platform === 'ios' || platform === 'all') {
  patchIos()
}
