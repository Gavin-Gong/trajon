import { Alert, Button, Card, Flex, Input, Space, Typography } from "antd";
import { useState } from "react";

import { runDevToolSql } from "../services/devToolsSqlRunner";
import { panelCardStyle, shellCardStyle } from "./PanelStyles";

const { Paragraph, Text, Title } = Typography;

const DEFAULT_SQL = "SELECT name FROM sqlite_master WHERE type = 'table' ORDER BY name;";

export function DevToolSection() {
  const [sql, setSql] = useState(DEFAULT_SQL);
  const [loading, setLoading] = useState(false);
  const [resultText, setResultText] = useState<string>();
  const [errorText, setErrorText] = useState<string>();

  async function handleRun() {
    setLoading(true);
    setErrorText(undefined);

    try {
      const result = await runDevToolSql(sql);
      setResultText(JSON.stringify(result, null, 2));
    } catch (error) {
      setResultText(undefined);
      setErrorText(error instanceof Error ? error.message : "执行 SQL 失败");
    } finally {
      setLoading(false);
    }
  }

  function handleClear() {
    setSql("");
    setResultText(undefined);
    setErrorText(undefined);
  }

  return (
    <Space orientation="vertical" size={20} style={{ width: "100%" }}>
      <Card style={shellCardStyle}>
        <Space orientation="vertical" size={12} style={{ width: "100%" }}>
          <Title level={4} style={{ margin: 0 }}>
            执行 SQLite SQL
          </Title>
          <Paragraph style={{ margin: 0, color: "rgba(0, 0, 0, 0.45)" }}>
            这个页面只在开发环境可见，用于直接查询或修改本地 SQLite 数据库。
          </Paragraph>
          <Alert
            type="warning"
            showIcon
            title="危险操作"
            description="这里执行的 SQL 会直接操作本地 SQLite 数据，包含 UPDATE、DELETE、DROP 等写入行为。"
          />
        </Space>
      </Card>

      <Card style={panelCardStyle}>
        <Space orientation="vertical" size={12} style={{ width: "100%" }}>
          <Text strong style={{ color: "rgba(0, 0, 0, 0.88)" }}>
            SQL
          </Text>
          <Input.TextArea
            value={sql}
            onChange={(event) => setSql(event.target.value)}
            rows={10}
            spellCheck={false}
            placeholder="输入要执行的 SQL"
          />
          <Flex gap={12}>
            <Button type="primary" loading={loading} onClick={() => void handleRun()}>
              执行
            </Button>
            <Button onClick={handleClear} disabled={loading}>
              清空
            </Button>
          </Flex>
        </Space>
      </Card>

      <Card style={panelCardStyle}>
        <Space orientation="vertical" size={12} style={{ width: "100%" }}>
          <Text strong style={{ color: "rgba(0, 0, 0, 0.88)" }}>
            结果 JSON
          </Text>
          {errorText ? <Alert type="error" showIcon title="执行失败" description={errorText} /> : null}
          <pre
            style={{
              margin: 0,
              padding: 12,
              overflowX: "auto",
              whiteSpace: "pre-wrap",
              wordBreak: "break-word",
              background: "rgba(15, 23, 42, 0.04)",
            }}
          >
            {resultText ?? '{\n  "message": "运行 SQL 后会在这里显示结果"\n}'}
          </pre>
        </Space>
      </Card>
    </Space>
  );
}
