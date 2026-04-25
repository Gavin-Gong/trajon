import { DeleteOutlined, EyeOutlined, UploadOutlined } from "@ant-design/icons";
import { Button, Card, Descriptions, Divider, Empty, Flex, Image, Modal, Space, Tag, Typography } from "antd";
import { useState } from "react";

import { netPosition, type JournalImage, type TradeJournal } from "../domain/journal";
import {
  isJournalImageManagementAvailable,
  pickJournalImageFiles,
  resolveJournalImageSrc,
} from "../platform/journalImages";
import { openPanelWindow } from "../platform/panelWindow";
import { TimelineSection } from "./TimelineSection";
import { tagColorForVariant } from "./EventPresentation";
import { formatJournalDate, formatNumber } from "./JournalFormatters";
import type { TradingJournalStore } from "./JournalStoreTypes";
import { dashedShellCardStyle, shellCardStyle } from "./PanelStyles";

const { Paragraph, Text, Title } = Typography;

export function JournalDetailPanel({
  journal,
  store,
}: {
  journal: TradeJournal | undefined;
  store?: TradingJournalStore;
}) {
  const [imageUrls, setImageUrls] = useState<Record<string, string>>({});

  if (!journal) {
    return (
      <Card style={dashedShellCardStyle}>
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description={
            <Space orientation="vertical" size={4}>
              <Text strong style={{ color: "rgba(0, 0, 0, 0.88)" }}>Journal 详情</Text>
              <Text style={{ color: "rgba(0, 0, 0, 0.45)" }}>
                从左侧日历选择一条 journal，这里会展示时间线与总结。
              </Text>
            </Space>
          }
        />
      </Card>
    );
  }

  const canManageImages = Boolean(store) && isJournalImageManagementAvailable();

  const ensureImageUrl = async (image: JournalImage) => {
    if (imageUrls[image.id]) {
      return imageUrls[image.id];
    }

    const nextUrl = await resolveJournalImageSrc(image.relativePath);
    setImageUrls((current) => ({ ...current, [image.id]: nextUrl }));
    return nextUrl;
  };

  return (
    <Card style={{ ...shellCardStyle, height: "fit-content" }}>
      <Space orientation="vertical" size={20} style={{ width: "100%" }}>
        <Flex justify="space-between" align="start" gap={12}>
          <Space orientation="vertical" size={8}>
            <Flex align="center" gap={8} wrap="wrap">
              <Title level={5} style={{ margin: 0, color: "rgba(0, 0, 0, 0.88)" }}>
                {journal.symbol || "未命名"}
              </Title>
              <Tag color={tagColorForVariant(journal.status === "open" ? "accent" : "neutral")}>
                {journal.status === "open" ? "进行中" : "已完成"}
              </Tag>
            </Flex>
            <Text style={{ color: "rgba(0, 0, 0, 0.45)" }}>
              {journal.market || "未标记市场"} · {formatJournalDate(journal.createdAt)}
            </Text>
          </Space>
          <Tag color={tagColorForVariant("neutral")}>净仓 {formatNumber(netPosition(journal.events))}</Tag>
        </Flex>

        <Descriptions
          size="small"
          column={1}
          items={[
            { key: "events", label: "事件数量", children: `${journal.events.length} 条` },
            {
              key: "tags",
              label: "标签",
              children:
                journal.tags.length > 0 ? (
                  <Space wrap size={[6, 6]}>
                    {journal.tags.map((tag) => (
                      <Tag key={tag}>{tag}</Tag>
                    ))}
                  </Space>
                ) : (
                  "-"
                ),
            },
            {
              key: "status",
              label: "状态",
              children: journal.status === "open" ? "进行中" : "已完成",
            },
          ]}
        />

        {journal.summary ? (
          <Paragraph style={{ margin: 0, color: "rgba(0, 0, 0, 0.45)", lineHeight: 1.8 }}>{journal.summary}</Paragraph>
        ) : null}

        <Space orientation="vertical" size={12} style={{ width: "100%" }}>
          <Flex justify="space-between" align="center" gap={12} wrap="wrap">
            <Text strong>复盘图片</Text>
            <Button
              type="default"
              shape="circle"
              icon={<UploadOutlined />}
              aria-label="上传图片"
              disabled={!canManageImages}
              onClick={async () => {
                if (!store) {
                  return;
                }

                const files = await pickJournalImageFiles();
                if (files.length === 0) {
                  return;
                }

                await store.actions.addJournalImages(journal.id, files);
              }}
            />
          </Flex>
          {!canManageImages ? <Text type="secondary">仅桌面端支持本地图片管理。</Text> : null}
          {journal.images.length === 0 ? (
            <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="还没有复盘图片" />
          ) : (
            <Flex wrap="wrap" gap={12}>
              {journal.images.map((image) => {
                const currentSrc = imageUrls[image.id];

                return (
                  <div key={image.id} style={{ position: "relative", width: 140 }}>
                    <Button
                      type="link"
                      aria-label={`预览图片 ${image.fileName}`}
                      style={{ padding: 0, height: "auto", borderRadius: 10, overflow: "hidden" }}
                      onMouseEnter={(event) => {
                        const overlay = event.currentTarget.querySelector<HTMLElement>(".journal-image-preview-overlay");
                        if (overlay) {
                          overlay.style.opacity = "1";
                        }
                      }}
                      onMouseLeave={(event) => {
                        const overlay = event.currentTarget.querySelector<HTMLElement>(".journal-image-preview-overlay");
                        if (overlay) {
                          overlay.style.opacity = "0";
                        }
                      }}
                      onClick={async () => {
                        const src = await ensureImageUrl(image);
                        Modal.info({
                          title: null,
                          width: 720,
                          icon: null,
                          content: <Image alt={image.fileName} src={src} style={{ width: "100%" }} />,
                          okText: "关闭",
                        });
                      }}
                    >
                      <div
                        style={{
                          width: 140,
                          height: 100,
                          borderRadius: 12,
                          overflow: "hidden",
                          background: "rgba(15, 23, 42, 0.04)",
                          border: "1px solid rgba(15, 23, 42, 0.08)",
                          display: "flex",
                          alignItems: "center",
                          justifyContent: "center",
                        }}
                      >
                        {currentSrc ? (
                          <div style={{ position: "relative", width: "100%", height: "100%" }}>
                            <img
                              alt={image.fileName}
                              src={currentSrc}
                              style={{ width: "100%", height: "100%", objectFit: "cover" }}
                            />
                            <div
                              className="journal-image-preview-overlay"
                              style={{
                                position: "absolute",
                                inset: 0,
                                display: "flex",
                                alignItems: "center",
                                justifyContent: "center",
                                background: "rgba(15, 23, 42, 0.24)",
                                color: "#fff",
                                opacity: 0,
                                transition: "opacity 160ms ease",
                              }}
                            >
                              <EyeOutlined style={{ fontSize: 20 }} />
                            </div>
                          </div>
                        ) : (
                          <EyeOutlined style={{ fontSize: 20, color: "rgba(15, 23, 42, 0.45)" }} />
                        )}
                      </div>
                    </Button>
                    <Button
                      aria-label={`删除图片 ${image.fileName}`}
                      shape="circle"
                      size="small"
                      icon={<DeleteOutlined />}
                      style={{
                        position: "absolute",
                        top: 8,
                        right: 8,
                        background: "rgba(255, 77, 79, 0.12)",
                        borderColor: "rgba(255, 77, 79, 0.18)",
                        color: "rgb(211, 47, 47)",
                      }}
                      onMouseEnter={(event) => {
                        event.currentTarget.style.background = "rgba(255, 77, 79, 0.2)";
                        event.currentTarget.style.borderColor = "rgba(255, 77, 79, 0.32)";
                        event.currentTarget.style.color = "rgb(255, 77, 79)";
                      }}
                      onMouseLeave={(event) => {
                        event.currentTarget.style.background = "rgba(255, 77, 79, 0.12)";
                        event.currentTarget.style.borderColor = "rgba(255, 77, 79, 0.18)";
                        event.currentTarget.style.color = "rgb(211, 47, 47)";
                      }}
                      onClick={() => {
                        if (!store) {
                          return;
                        }

                        void store.actions.removeJournalImage(journal.id, image.id);
                      }}
                      onMouseDown={(event) => {
                        event.stopPropagation();
                      }}
                    />
                  </div>
                );
              })}
            </Flex>
          )}
        </Space>

        <Button
          type="default"
          onClick={() => {
            void openPanelWindow({ journalId: journal.id });
          }}
        >
          继续编辑这条 Journal
        </Button>

        <Divider style={{ margin: 0 }} />

        <TimelineSection journal={journal} title="Journal 时间线" />
      </Space>
    </Card>
  );
}
