import fs from 'node:fs'
import path from 'node:path'

const platform = (process.argv[2] || 'all').toLowerCase()
const cwd = process.cwd()
const mobileRoot = path.basename(cwd) === 'mobile' ? cwd : path.join(cwd, 'mobile')
const appId = 'app.rpbox.mobile'

function ensureFile(filePath, contents) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true })
  fs.writeFileSync(filePath, contents, 'utf8')
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
}

function patchIos() {
  const infoPlistPath = path.join(mobileRoot, 'ios', 'App', 'App', 'Info.plist')
  if (!fs.existsSync(infoPlistPath)) return

  let plist = fs.readFileSync(infoPlistPath, 'utf8')
  if (!plist.includes('<key>CFBundleURLTypes</key>')) {
    const block = `
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
	</array>
`
    plist = plist.replace(/<\/dict>\s*<\/plist>\s*$/, `${block}</dict>\n</plist>\n`)
    fs.writeFileSync(infoPlistPath, plist, 'utf8')
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
