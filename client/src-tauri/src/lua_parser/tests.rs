use super::parser::parse_variable_from_str;
use serde_json::json;

#[test]
fn test_parse_simple_table() {
    let lua = r#"
        TestVar = {
            ["key1"] = "value1",
            ["key2"] = 123,
        }
    "#;
    let result = parse_variable_from_str(lua, "TestVar").unwrap();
    assert_eq!(result["key1"], "value1");
    assert_eq!(result["key2"], 123.0);
}

#[test]
fn test_parse_nested_table() {
    let lua = r#"
        TestVar = {
            ["outer"] = {
                ["inner"] = "nested",
            },
        }
    "#;
    let result = parse_variable_from_str(lua, "TestVar").unwrap();
    assert_eq!(result["outer"]["inner"], "nested");
}

#[test]
fn test_parse_chinese_content() {
    let lua = r#"
        TRP3_Profiles = {
            ["test"] = {
                ["profileName"] = "芙拉莉雅",
            },
        }
    "#;
    let result = parse_variable_from_str(lua, "TRP3_Profiles").unwrap();
    assert_eq!(result["test"]["profileName"], "芙拉莉雅");
}

#[test]
fn test_parse_escape_sequences() {
    let lua = r#"
        TestVar = {
            ["text"] = "line1\nline2\ttab",
        }
    "#;
    let result = parse_variable_from_str(lua, "TestVar").unwrap();
    assert_eq!(result["text"], "line1\nline2\ttab");
}

#[test]
fn test_parse_boolean_and_nil() {
    let lua = r#"
        TestVar = {
            ["active"] = true,
            ["disabled"] = false,
            ["empty"] = nil,
        }
    "#;
    let result = parse_variable_from_str(lua, "TestVar").unwrap();
    assert_eq!(result["active"], true);
    assert_eq!(result["disabled"], false);
    assert!(result["empty"].is_null());
}

#[test]
fn test_parse_array() {
    let lua = r#"
        TestVar = { "a", "b", "c" }
    "#;
    let result = parse_variable_from_str(lua, "TestVar").unwrap();
    assert_eq!(result[0], "a");
    assert_eq!(result[1], "b");
    assert_eq!(result[2], "c");
}

#[test]
fn test_parse_with_comments() {
    let lua = r#"
        -- This is a comment
        TestVar = {
            ["key"] = "value", -- inline comment
        }
    "#;
    let result = parse_variable_from_str(lua, "TestVar").unwrap();
    assert_eq!(result["key"], "value");
}

#[test]
fn test_variable_not_found() {
    let lua = r#"OtherVar = {}"#;
    let result = parse_variable_from_str(lua, "TestVar");
    assert!(result.is_err());
}
