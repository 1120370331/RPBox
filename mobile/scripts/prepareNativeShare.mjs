import fs from 'node:fs'
import path from 'node:path'

const platform = (process.argv[2] || 'all').toLowerCase()
const cwd = process.cwd()
const mobileRoot = path.basename(cwd) === 'mobile' ? cwd : path.join(cwd, 'mobile')
const appId = 'app.rpbox.mobile'
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
  const appLinkBlock = buildAndroidAppLinkBlock()

  if (/<!-- RPBOX_APP_LINKS_START -->[\s\S]*<!-- RPBOX_APP_LINKS_END -->/.test(manifest)) {
    manifest = manifest.replace(/<!-- RPBOX_APP_LINKS_START -->[\s\S]*<!-- RPBOX_APP_LINKS_END -->/, appLinkBlock)
  } else if (/<\/activity>/.test(manifest)) {
    manifest = manifest.replace(/<\/activity>/, `${appLinkBlock}\n        </activity>`)
  }

  fs.writeFileSync(manifestPath, manifest, 'utf8')
}

function patchIos() {
  const infoPlistPath = path.join(mobileRoot, 'ios', 'App', 'App', 'Info.plist')
  if (!fs.existsSync(infoPlistPath)) return

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
  if (fs.existsSync(pbxprojPath)) {
    let pbxproj = fs.readFileSync(pbxprojPath, 'utf8')
    if (/CODE_SIGN_ENTITLEMENTS = [^;]+;/.test(pbxproj)) {
      pbxproj = pbxproj.replace(/CODE_SIGN_ENTITLEMENTS = [^;]+;/g, 'CODE_SIGN_ENTITLEMENTS = App/App.entitlements;')
    } else {
      pbxproj = pbxproj.replace(/INFOPLIST_FILE = App\/Info\.plist;/g, 'INFOPLIST_FILE = App/Info.plist;\n\t\t\t\tCODE_SIGN_ENTITLEMENTS = App/App.entitlements;')
    }
    fs.writeFileSync(pbxprojPath, pbxproj, 'utf8')
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
