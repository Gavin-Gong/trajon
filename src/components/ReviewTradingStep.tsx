import { DeleteOutlined, EyeOutlined, UploadOutlined } from "@ant-design/icons";
import { Button, Empty, Flex, Image, Input, Modal, Select, Space, Typography } from "antd";
import { useEffect, useMemo, useRef, useState } from "react";

import type { JournalImage, TradeJournal } from "../domain/journal";
import {
  isJournalImageManagementAvailable,
  pickJournalImageFiles,
  resolveJournalImageSrc,
} from "../platform/journalImages";
import type { TradingJournalStore } from "./JournalStoreTypes";
import { plainPanelSectionStyle } from "./PanelStyles";

const { Text, Title } = Typography;

export function ReviewTradingStep({
  journal,
  store,
  summary,
  tags,
  onSummaryChange,
  onTagsChange,
  onBack,
  onConfirm,
}: {
  journal: TradeJournal | undefined;
  store: TradingJournalStore;
  summary: string;
  tags: string[];
  onSummaryChange: (value: string) => void;
  onTagsChange: (value: string[]) => void;
  onBack?: () => void;
  onConfirm?: () => void;
}) {
  const [imageUrls, setImageUrls] = useState<Record<string, string>>({});
  const imageUrlsRef = useRef<Record<string, string>>({});
  const canManageImages = isJournalImageManagementAvailable();
  const historicalTagOptions = useMemo(
    () =>
      Array.from(
        store.state.journals
          .filter((item) => item.status === "done")
          .reduce((tagMap, item) => {
            for (const tag of item.tags) {
              const trimmedTag = tag.trim();
              if (!trimmedTag) {
                continue;
              }

              const normalizedTag = trimmedTag.toLowerCase();
              if (!tagMap.has(normalizedTag)) {
                tagMap.set(normalizedTag, trimmedTag);
              }
            }

            return tagMap;
          }, new Map<string, string>())
          .values(),
      ).map((tag) => ({ label: tag, value: tag })),
    [store.state.journals],
  );

  useEffect(() => {
    imageUrlsRef.current = imageUrls;
  }, [imageUrls]);

  useEffect(() => {
    let cancelled = false;

    async function preloadImageUrls(images: JournalImage[]) {
      const pendingImages = images.filter((image) => !imageUrlsRef.current[image.id]);
      if (pendingImages.length === 0) {
        return;
      }

      const entries = await Promise.all(
        pendingImages.map(async (image) => [image.id, await resolveJournalImageSrc(image.relativePath)] as const),
      );

      if (cancelled) {
        return;
      }

      setImageUrls((current) => ({
        ...current,
        ...Object.fromEntries(entries),
      }));
    }

    if (!journal || journal.images.length === 0) {
      return () => {
        cancelled = true;
      };
    }

    void preloadImageUrls(journal.images);

    return () => {
      cancelled = true;
    };
  }, [journal]);

  const ensureImageUrl = async (image: JournalImage) => {
    const existingSrc = imageUrlsRef.current[image.id];
    if (existingSrc) {
      return existingSrc;
    }

    const nextUrl = await resolveJournalImageSrc(image.relativePath);
    setImageUrls((current) => ({ ...current, [image.id]: nextUrl }));
    return nextUrl;
  };

  return (
    <div style={plainPanelSectionStyle}>
      <Space orientation="vertical" size={16} style={{ width: "100%" }}>
        <Space orientation="vertical" size={4}>
          <Title level={5} style={{ margin: 0, color: "rgba(0, 0, 0, 0.88)" }}>
            复盘总结
          </Title>
          <Text style={{ color: "rgba(0, 0, 0, 0.45)" }}>
            记录这笔交易的复盘结论、标签和截图。
          </Text>
        </Space>

        <Space orientation="vertical" size={16} style={{ width: "100%" }}>
          <label style={{ width: "100%" }}>
            <Space orientation="vertical" size={8} style={{ width: "100%" }}>
              <Text strong>复盘</Text>
              <Input.TextArea
                aria-label="复盘"
                rows={5}
                placeholder="可选：这笔交易哪里做得好，哪里需要改进？"
                value={summary}
                onChange={(event) => onSummaryChange(event.currentTarget.value)}
              />
            </Space>
          </label>

          <label style={{ width: "100%" }}>
            <Space orientation="vertical" size={8} style={{ width: "100%" }}>
              <Text strong>标签</Text>
              <Select
                aria-label="标签"
                data-testid="review-tags-select"
                mode="tags"
                placeholder="输入后回车，可添加多个标签"
                options={historicalTagOptions}
                tokenSeparators={[","]}
                value={tags}
                onChange={(value) => onTagsChange(value.map((item) => String(item)))}
              />
            </Space>
          </label>

          <Space orientation="vertical" size={12} style={{ width: "100%" }}>
            <Space style={{ width: "100%", justifyContent: "space-between" }} wrap>
              <Text strong>复盘图片</Text>
              <Button
                type="default"
                shape="circle"
                icon={<UploadOutlined />}
                aria-label="上传图片"
                disabled={!journal || !canManageImages}
                onClick={async () => {
                  if (!journal) {
                    return;
                  }

                  const files = await pickJournalImageFiles();
                  if (files.length === 0) {
                    return;
                  }

                  await store.actions.addJournalImages(journal.id, files);
                }}
              />
            </Space>

            {!canManageImages ? <Text type="secondary">仅桌面端支持本地图片管理。</Text> : null}

            {!journal || journal.images.length === 0 ? (
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="还没有复盘图片" />
            ) : (
              <Space wrap size={12}>
                {journal.images.map((image) => {
                  const currentSrc = imageUrls[image.id];

                  return (
                    <div key={image.id} style={{ position: "relative", width: 140 }}>
                      <Button
                        type="link"
                        aria-label={`预览图片 ${image.fileName}`}
                        style={{
                          padding: 0,
                          height: "auto",
                          borderRadius: 10,
                          overflow: "hidden",
                        }}
                        onMouseEnter={(event) => {
                          const overlay = event.currentTarget.querySelector<HTMLElement>(
                            ".journal-image-preview-overlay",
                          );
                          if (overlay) {
                            overlay.style.opacity = "1";
                          }
                        }}
                        onMouseLeave={(event) => {
                          const overlay = event.currentTarget.querySelector<HTMLElement>(
                            ".journal-image-preview-overlay",
                          );
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
                          if (!journal) {
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
              </Space>
            )}
          </Space>
        </Space>

        {onBack || onConfirm ? (
          <Flex justify="space-between" gap={12} wrap>
            <Button onClick={onBack}>返回记录</Button>
            <Button type="primary" onClick={onConfirm}>
              完成复盘
            </Button>
          </Flex>
        ) : null}
      </Space>
    </div>
  );
}
