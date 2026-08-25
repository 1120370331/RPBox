import assert from 'node:assert/strict'
import { spawnSync } from 'node:child_process'
import { createHash } from 'node:crypto'
import fs from 'node:fs'
import test from 'node:test'
import { parse } from 'yaml'

const workflow = fs.readFileSync(
  new URL('../../.github/workflows/app-store-submit.yml', import.meta.url),
  'utf8',
)
const prepareWorkflow = fs.readFileSync(
  new URL('../../.github/workflows/app-store-prepare.yml', import.meta.url),
  'utf8',
)
const releaseNotes = fs.readFileSync(
  new URL('../release-notes/1.1.txt', import.meta.url),
  'utf8',
).replaceAll('\r\n', '\n').trim()
const workflowDocument = parse(workflow)
const prepareDocument = parse(prepareWorkflow)

function embeddedPythonBlocks(source) {
  return [...source.matchAll(/python3 - <<'PY'\r?\n([\s\S]*?)\r?\n {10}PY/g)].map((match) =>
    match[1]
      .split(/\r?\n/)
      .map((line) => (line.startsWith('          ') ? line.slice(10) : line))
      .join('\n'),
  )
}

function topLevelAssignment(source, name) {
  const lines = source.split('\n')
  const start = lines.findIndex((line) => line.startsWith(`${name} =`))
  assert.notEqual(start, -1, `missing Python assignment ${name}`)
  let end = start + 1
  while (end < lines.length && !/^[A-Z][A-Z0-9_]* = /.test(lines[end])) end += 1
  return lines.slice(start, end).join('\n').trimEnd()
}

function dedentedLiteral(assignment) {
  const match = assignment.match(/textwrap\.dedent\((?:f)?"""\\\n([\s\S]*?)\n"""\)\.strip\(\)/)
  assert.ok(match, 'expected a textwrap.dedent triple-quoted literal')
  return match[1].replaceAll('\r\n', '\n').trim()
}

test('submission workflow freezes the release contract and GitHub safety gates', () => {
  const dispatch = workflowDocument.on.workflow_dispatch
  assert.equal(dispatch.inputs.apply.required, true)
  assert.equal(dispatch.inputs.apply.default, false)
  assert.equal(dispatch.inputs.apply.type, 'boolean')
  assert.equal(dispatch.inputs.confirmation.required, true)
  assert.equal(dispatch.inputs.confirmation.type, 'string')
  assert.deepEqual(workflowDocument.permissions, { contents: 'read' })
  assert.deepEqual(workflowDocument.concurrency, {
    group: 'app-store-submit-1-1',
    'cancel-in-progress': false,
  })
  assert.match(workflow, /GITHUB_REF[^\n]+refs\/heads\/main/)
  assert.match(workflow, /ASC_EXPECTED_APP_ID: '6761112311'/)
  assert.match(workflow, /ASC_EXPECTED_BUNDLE_ID: app\.rpbox\.mobile/)
  assert.match(workflow, /ASC_EXPECTED_VERSION: '1\.1'/)
  assert.match(workflow, /ASC_EXPECTED_VERSION_STATE: PREPARE_FOR_SUBMISSION/)
  assert.match(workflow, /ASC_EXPECTED_BUILD_NUMBER: '1000042'/)
  assert.match(workflow, /ASC_EXPECTED_BUILD_STATE: VALID/)
  assert.match(workflow, /ASC_EXPECTED_RELEASE_TYPE: AFTER_APPROVAL/)
  assert.match(workflow, /uses: actions\/checkout@v4/)
  assert.match(workflow, /ref: \$\{\{ github\.sha \}\}/)
  assert.match(workflow, /submodules: false/)
  assert.match(workflow, /name: Remove checkout credentials/)
  assert.match(workflow, /git config --local --unset-all http\.https:\/\/github\.com\/\.extraheader/)
})

test('every embedded Python block compiles without executing network code', () => {
  const blocks = embeddedPythonBlocks(workflow)
  assert.ok(blocks.length > 0)
  const python = process.platform === 'win32' ? 'python' : 'python3'
  for (const [index, source] of blocks.entries()) {
    const result = spawnSync(
      python,
      [
        '-c',
        'import sys; compile(sys.stdin.buffer.read().decode("utf-8"), "embedded-workflow.py", "exec")',
      ],
      { input: Buffer.from(source, 'utf8'), encoding: 'utf8' },
    )
    assert.equal(result.status, 0, `embedded Python block ${index + 1} failed: ${result.stderr}`)
  }
})

test('checked-out repository release contract fails closed before any ASC client is created', () => {
  const contractCheck = workflow.indexOf('release_contract_path = Path("mobile/ios/release.json")')
  const clientCreation = workflow.indexOf('client = AscClient(TokenProvider())')
  assert.ok(contractCheck > 0 && clientCreation > contractCheck)

  const preflight = workflow.slice(contractCheck, clientCreation)
  assert.match(preflight, /type\(release_contract\) is not dict/)
  assert.match(preflight, /set\(release_contract\) != \{/)
  assert.match(preflight, /"version",\s*\n\s+"buildNumber",/)
  assert.match(preflight, /type\(release_contract\.get\("version"\)\) is not str/)
  assert.match(preflight, /release_contract\["version"\] != "1\.1"/)
  assert.match(preflight, /type\(release_contract\.get\("buildNumber"\)\) is not int/)
  assert.match(preflight, /release_contract\["buildNumber"\] != 1000042/)
  assert.match(preflight, /Path\("mobile\/ios\/App\/App\.xcodeproj\/project\.pbxproj"\)/)
  assert.match(preflight, /len\(project_build_numbers\) != 2/)
  assert.match(preflight, /value\.strip\(\) != "1000042"/)
})

test('apply requires a second exact confirmation while dry run cannot mutate', () => {
  const phrase = 'SUBMIT-RPBOX-IOS-1.1-BUILD-1000042'
  assert.match(workflow, new RegExp(`ASC_CONFIRMATION_PHRASE: ${phrase}`))
  assert.match(workflow, /if \[ "\$APPLY" = "true" \] && \[ "\$CONFIRMATION" != "\$ASC_CONFIRMATION_PHRASE" \]/)
  assert.match(workflow, /if APPLY and CONFIRMATION != CONFIRMATION_PHRASE:/)
  assert.match(workflow, /if not APPLY:\s*\n\s+raise RuntimeError\("Mutation attempted during dry run"\)/)

  const dryRunBranch = workflow.lastIndexOf('if not APPLY:')
  const dryRunReturn = workflow.indexOf('return', dryRunBranch)
  const firstMainMutation = workflow.indexOf('submission = reusable_submission', dryRunReturn)
  assert.ok(dryRunBranch > 0 && dryRunReturn > dryRunBranch)
  assert.ok(firstMainMutation > dryRunReturn)
  assert.match(workflow.slice(dryRunBranch, dryRunReturn), /No mutation was sent/)
})

test('preflight validates all frozen App Store Connect submission prerequisites', () => {
  for (const expected of [
    '/v1/apps/{APP_ID}/appStoreVersions',
    '/v1/builds',
    '/v1/builds/{build_id}/preReleaseVersion',
    '/v1/appStoreVersions/{version_id}/build',
    '/v1/appStoreVersions/{version_id}/appStoreVersionLocalizations',
    '/v1/appStoreVersions/{version_id}/appStoreReviewDetail',
    '/v1/appStoreVersionLocalizations/{localization_id}/appScreenshotSets',
    '/v1/appScreenshotSets/{screenshot_set_id}/appScreenshots',
    '/v1/appInfos/{current_app_info_id}/ageRatingDeclaration',
    '/v1/apps/{APP_ID}/reviewSubmissions',
  ]) {
    assert.ok(workflow.includes(expected), `missing endpoint contract ${expected}`)
  }
  for (const field of [
    'contactFirstName',
    'contactLastName',
    'contactPhone',
    'contactEmail',
    'demoAccountName',
    'demoAccountPassword',
  ]) {
    assert.ok(workflow.includes(`"${field}"`), `missing protected review field ${field}`)
  }
  assert.match(workflow, /review_attributes\.get\("demoAccountRequired"\) is not True/)
  assert.match(workflow, /notes != REVIEW_NOTES or placeholder\.search/)
})

test('approved metadata and release notes exactly match the prepare workflow', () => {
  const submitMain = embeddedPythonBlocks(workflow).find((source) => source.includes('DESCRIPTION = ('))
  const prepareMain = embeddedPythonBlocks(prepareWorkflow).find((source) => source.includes('DESCRIPTION = ('))
  assert.ok(submitMain && prepareMain)

  for (const name of ['DESCRIPTION', 'WHATS_NEW', 'REVIEW_NOTES']) {
    assert.equal(
      topLevelAssignment(submitMain, name),
      topLevelAssignment(prepareMain, name),
      `${name} drifted from app-store-prepare.yml`,
    )
  }
  assert.equal(dedentedLiteral(topLevelAssignment(submitMain, 'WHATS_NEW')), releaseNotes)
  for (const name of ['PUBLIC_SUPPORT_URL', 'PUBLIC_MARKETING_URL', 'PUBLIC_PRIVACY_URL']) {
    assert.equal(workflowDocument.env[name], prepareDocument.env[name], `${name} drifted`)
  }

  assert.match(workflow, /localization_attributes\.get\(field\) != expected/)
  assert.match(workflow, /release_notes != WHATS_NEW/)
  assert.match(workflow, /KEYWORDS_LENGTH = 19/)
  assert.match(
    workflow,
    /KEYWORDS_SHA256 = "5a69f9891492a90c7bfd351d5c118a2a07f67375d72c283c3cf523328c7da176"/,
  )
  assert.match(workflow, /len\(keywords\) != KEYWORDS_LENGTH/)
  assert.match(workflow, /hashlib\.sha256\(keywords\.encode\("utf-8"\)\)\.hexdigest\(\) != KEYWORDS_SHA256/)
  assert.doesNotMatch(workflow, /f["'].*\{keywords\}/)
})

test('iPhone and iPad screenshots use exact ordered approved contracts', () => {
  const expectedNames = [
    '01-guild-plaza.png',
    '02-story-management.png',
    '03-resource-market.png',
    '04-rp-community.png',
  ]
  const screenshotDirectory = new URL('../../artifacts/app-store-screenshots-1242x2688/', import.meta.url)
  const actualNames = fs.readdirSync(screenshotDirectory).sort()
  assert.deepEqual(actualNames, expectedNames)

  for (const [index, name] of expectedNames.entries()) {
    const payload = fs.readFileSync(new URL(name, screenshotDirectory))
    const sha256 = createHash('sha256').update(payload).digest('hex')
    const variable = `SCREENSHOT_SHA256_0${index + 1}`
    assert.equal(workflowDocument.env[variable], sha256)
    assert.equal(workflowDocument.env[variable], prepareDocument.env[variable])
    assert.equal(payload.subarray(0, 8).toString('hex'), '89504e470d0a1a0a')
    assert.equal(payload.subarray(12, 16).toString('ascii'), 'IHDR')
    assert.equal(payload.readUInt32BE(16), 1242)
    assert.equal(payload.readUInt32BE(20), 2688)
  }

  const ipadContract = [
    ['bb5683e30c1bcb71bd77b2cfc3b7d0ba_2732x2048.jpg', 533634, '757c163aaadb21965fb8500e47e4d56d'],
    ['f23c33aed97c29f10000f4df229e0edb_2732x2048.jpg', 658259, '368aa71ecae380ec7cb6746b95d42ca0'],
    ['b1fb3442fd2dc8f9eb18c8577b7c70ca_2732x2048.png', 2503979, 'f3ce75507d22aa30f41cc725cdaf55aa'],
    ['1cf03c055edc5c82bab9a195a495361a_2732x2048.png', 594484, 'e5b0080420058fa645ad16e0790ae121'],
  ]
  const ipadAssignment = topLevelAssignment(
    embeddedPythonBlocks(workflow)[0],
    'IPAD_APPROVED_SCREENSHOTS',
  )
  for (const [name, size, md5] of ipadContract) {
    assert.ok(ipadAssignment.includes(`"${name}"`))
    assert.ok(ipadAssignment.includes(String(size)))
    assert.ok(ipadAssignment.includes(`"${md5}"`))
  }
  assert.equal((ipadAssignment.match(/2732,/g) || []).length, 4)
  assert.equal((ipadAssignment.match(/2048,/g) || []).length, 4)

  assert.match(workflow, /len\(screenshots\) != 4 or len\(approved\) != 4/)
  assert.match(workflow, /zip\(screenshots, approved\)/)
  for (const field of ['fileName', 'fileSize', 'sourceFileChecksum', 'imageAsset']) {
    assert.ok(workflow.includes(`"${field}"`), `missing screenshot field ${field}`)
  }
  assert.match(workflow, /screenshot_state\(screenshot\) != "COMPLETE"/)
  assert.match(workflow, /Screenshot contracts: exact approved iPhone and iPad inventories verified/)
})

test('age rating is tied to the unique current appInfo and exact approved values', () => {
  assert.match(workflow, /attributes\(item\)\.get\("appStoreState"\) == app_version_state/)
  assert.match(workflow, /len\(current_app_infos\) != 1/)
  const booleanFields = {
    userGeneratedContent: true,
    messagingAndChat: true,
    unrestrictedWebAccess: false,
    advertising: false,
    gambling: false,
    lootBox: false,
  }
  for (const [field, expected] of Object.entries(booleanFields)) {
    assert.match(workflow, new RegExp(`"${field}": ${expected ? 'True' : 'False'}`))
  }
  for (const field of [
    'alcoholTobaccoOrDrugUseOrReferences',
    'contests',
    'gamblingSimulated',
    'horrorOrFearThemes',
    'matureOrSuggestiveThemes',
    'medicalOrTreatmentInformation',
    'profanityOrCrudeHumor',
    'sexualContentGraphicAndNudity',
    'sexualContentOrNudity',
    'violenceCartoonOrFantasy',
    'violenceRealistic',
    'violenceRealisticProlongedGraphicOrSadistic',
  ]) {
    assert.ok(workflow.includes(`"${field}"`), `missing NONE age-rating field ${field}`)
  }
  assert.match(workflow, /age_rating_attributes\.get\(field\) != "NONE"/)
})

test('workflow uses only the modern reviewSubmissions API and exact JSON:API types', () => {
  assert.match(workflow, /"POST",\s*\n\s+"\/v1\/reviewSubmissions"/)
  assert.match(workflow, /"type": "reviewSubmissions"/)
  assert.match(workflow, /"POST",\s*\n\s+"\/v1\/reviewSubmissionItems"/)
  assert.match(workflow, /"type": "reviewSubmissionItems"/)
  assert.match(workflow, /"appStoreVersion": \{\s*\n\s+"data": \{"type": "appStoreVersions", "id": version_id\}/)
  assert.match(workflow, /"PATCH",\s*\n\s+f"\/v1\/reviewSubmissions\/\{submission_id\}"/)
  assert.match(workflow, /"attributes": \{"submitted": True\}/)
  assert.match(workflow, /Refused a mutation that does not match the frozen JSON:API contract/)
  assert.match(workflow, /data\.get\("attributes"\) == \{"submitted": True\}/)
  assert.doesNotMatch(workflow, /appStoreVersionSubmissions/)
  assert.doesNotMatch(workflow, /\/appStoreVersionSubmission(?:\W|$)/)
  assert.doesNotMatch(
    workflow,
    /\/v1\/reviewSubmissionItems\/\{[^}]+\}\/appStoreVersion/,
  )
})

test('fresh and recovery states are strict and linkage uses version state plus one item', () => {
  const python = embeddedPythonBlocks(workflow)[0]
  const selector = python.slice(
    python.indexOf('def select_submission_path('),
    python.indexOf('def assert_exact_submission_item('),
  )
  const linkageWait = python.slice(
    python.indexOf('def wait_for_version_item_linkage('),
    python.indexOf('def create_submission_with_recovery('),
  )

  assert.match(workflow, /RECOVERY_VERSION_STATE = "READY_FOR_REVIEW"/)
  assert.match(workflow, /app_version_state not in \{VERSION_STATE, RECOVERY_VERSION_STATE\}/)
  assert.match(workflow, /def select_submission_path/)
  assert.match(workflow, /app_version_state == VERSION_STATE and not active/)
  assert.match(workflow, /return None, True, "fresh"/)
  assert.match(workflow, /if app_version_state == VERSION_STATE:/)
  assert.match(workflow, /if items:\s*\n\s+raise RuntimeError\("A fresh PREPARE_FOR_SUBMISSION version has an unexpected item"\)/)
  assert.match(workflow, /if app_version_state == RECOVERY_VERSION_STATE:/)
  assert.match(workflow, /if len\(items\) != 1:/)
  assert.match(selector, /resolve_submission_item_version_id\(client, items\[0\]\) != version_id/)
  assert.match(workflow, /return candidate, False, "recovery"/)
  assert.match(workflow, /candidate_attributes\.get\("platform"\) != "IOS"/)
  assert.match(workflow, /candidate_attributes\.get\("state"\) != "READY_FOR_REVIEW"/)
  assert.match(workflow, /def wait_for_version_item_linkage/)
  assert.ok(workflow.includes('f"/v1/appStoreVersions/{version_id}"'))
  assert.match(workflow, /version_state == RECOVERY_VERSION_STATE and len\(items\) == 1/)
  assert.match(linkageWait, /resolve_submission_item_version_id\(client, items\[0\]\) != version_id/)
  assert.match(workflow, /if submission_path != "recovery":/)
  assert.match(workflow, /Only a verified recovery path may reuse an existing item/)
})

test('item linkage resolves exactly through inline data or self include without forbidden relationship GET', () => {
  assert.match(workflow, /def get_resource_with_included/)
  assert.match(workflow, /included = document\.get\("included", \[\]\)/)
  assert.match(workflow, /def inline_item_version_id/)
  assert.match(workflow, /relationship = relationships\(item\)\.get\("appStoreVersion"\)/)
  assert.match(workflow, /data\.get\("type"\) != "appStoreVersions"/)
  assert.match(workflow, /def resolve_submission_item_version_id/)
  assert.ok(workflow.includes('f"/v1/reviewSubmissionItems/{item_id}"'))
  assert.match(workflow, /params=\{"include": "appStoreVersion"\}/)
  assert.match(workflow, /resource\.get\("type"\) == "appStoreVersions"/)
  assert.match(workflow, /if len\(included_versions\) != 1:/)
  assert.match(workflow, /if not candidate_ids or len\(set\(candidate_ids\)\) != 1:/)
  assert.match(workflow, /assert_exact_submission_item\(client, submission_id, version_id\)/)
  assert.doesNotMatch(
    workflow,
    /\/v1\/reviewSubmissionItems\/\{[^}]+\}\/appStoreVersion/,
  )
})

test('workflow recovers partial apply, polls authoritative state, and emits no sensitive artifact', () => {
  assert.match(workflow, /create_submission_with_recovery/)
  assert.match(workflow, /add_item_with_recovery/)
  assert.match(workflow, /wait_for_version_item_linkage/)
  assert.match(workflow, /WAITING_FOR_REVIEW/)
  assert.match(workflow, /IN_REVIEW/)
  assert.match(workflow, /poll_submitted_state/)
  assert.match(workflow, /time\.monotonic\(\)/)
  assert.match(workflow, /safe_error_message/)
  assert.match(workflow, /REDACTED_RESOURCE_ID/)
  assert.match(workflow, /values suppressed/)
  assert.doesNotMatch(workflow, /actions\/upload-artifact/)
  assert.doesNotMatch(workflow, /print\([^\n]*(?:ASC_ISSUER_ID|ASC_KEY_ID|ASC_API_KEY_BASE64|contactFirstName|demoAccountPassword)/)
})
