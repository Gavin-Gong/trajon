import { EditOutlined } from "@ant-design/icons";
import { Button, Flex, Image, Modal, Select, Space, Tag, Typography } from "antd";
import { useEffect, useMemo, useState } from "react";

import { netPosition, type JournalImage, type TradeJournal } from "../domain/journal";
import { resolveJournalImageSrc } from "../platform/journalImages";
import { formatNumber } from "./JournalFormatters";
import { tagColorForVariant } from "./EventPresentation";
import { plainMutedBlockStyle, plainPanelSectionStyle } from "./PanelStyles";

const { Paragraph, Text, Title } = Typography;

export function PanelJournalSummary({
  journal,
  journals,
  onUpdateSymbol,
}: {
  journal: TradeJournal | undefined;
  journals: TradeJournal[];
  onUpdateSymbol: (journalId: string, symbol: string) => Promise<void>;
}) {
  const [imageUrls, setImageUrls] = useState<Record<string, string>>({});
  const historySymbols = useMemo(() => {
    const seen = new Set<string>();

    return [...journals]
      .filter((item) => item.symbol.trim())
      .sort((left, right) => right.updatedAt.localeCompare(left.updatedAt))
      .filter((item) => {
        const symbol = item.symbol.trim();
        if (seen.has(symbol)) {
          return false;
        }

        seen.add(symbol);
        return true;
      })
      .map((item) => item.symbol.trim());
  }, [journals]);
  const recentSymbol = historySymbols[0] ?? "";
  const [isEditingSymbol, setIsEditingSymbol] = useState(() => !journal?.symbol.trim());
  const [symbolDraft, setSymbolDraft] = useState(() => journal?.symbol.trim() || recentSymbol);

  useEffect(() => {
    const nextSymbol = journal?.symbol.trim() || recentSymbol;
    setIsEditingSymbol(!journal?.symbol.trim());
    setSymbolDraft(nextSymbol);
  }, [journal?.id, journal?.symbol, recentSymbol]);

  const visibleImages = journal?.images.slice(0, 3) ?? [];
  const overflowImageCount = Math.max((journal?.images.length ?? 0) - visibleImages.length, 0);

  useEffect(() => {
    let cancelled = false;

    async function preloadVisibleImages(images: JournalImage[]) {
      const entries = await Promise.all(
        images.map(async (image) => [image.id, await resolveJournalImageSrc(image.relativePath)] as const),
      );

      if (cancelled) {
        return;
      }

      setImageUrls((current) => ({
        ...current,
        ...Object.fromEntries(entries),
      }));
    }

    if (visibleImages.length > 0) {
      void preloadVisibleImages(visibleImages);
    }

    return () => {
      cancelled = true;
    };
  }, [visibleImages]);

  const commitSymbol = async (nextSymbol: string) => {
    const trimmedSymbol = nextSymbol.trim();
    if (!journal || !trimmedSymbol) {
      setSymbolDraft(trimmedSymbol);
      return;
    }

    await onUpdateSymbol(journal.id, trimmedSymbol);
    setSymbolDraft(trimmedSymbol);
    setIsEditingSymbol(false);
  };

  const openImagePreview = async (image: JournalImage) => {
    const src = imageUrls[image.id] ?? (await resolveJournalImageSrc(image.relativePath));
    setImageUrls((current) => ({ ...current, [image.id]: src }));

    Modal.info({
      title: image.fileName,
      width: 720,
      icon: null,
      content: <Image alt={image.fileName} src={src} style={{ width: "100%" }} />,
      okText: "关闭",
    });
  };

  return (
    <div style={plainPanelSectionStyle}>
      <Space orientation="vertical" size={16} style={{ width: "100%" }}>
        {journal ? (
          <>
            <Flex justify="space-between" align="start" gap={12}>
              <Space orientation="vertical" size={8}>
                <Flex align="center" gap={8} wrap="wrap">
                  {isEditingSymbol ? (
                    <Select
                      aria-label="交易标的"
                      showSearch
                      autoFocus
                      placeholder="选择或输入交易标的"
                      searchValue={symbolDraft}
                      style={{ minWidth: 180 }}
                      filterOption={(input, option) => String(option?.value ?? "").includes(input.trim())}
                      options={historySymbols.map((symbol) => ({ label: symbol, value: symbol }))}
                      onChange={(value) => {
                        if (typeof value === "string" && value.trim()) {
                          void commitSymbol(value);
                        }
                      }}
                      onSearch={setSymbolDraft}
                      onBlur={() => {
                        void commitSymbol(symbolDraft);
                      }}
                      onInputKeyDown={(event) => {
                        if (event.key !== "Enter") {
                          return;
                        }

                        event.preventDefault();
                        void commitSymbol(symbolDraft);
                      }}
                    />
                  ) : (
                    <>
                      <Title level={5} style={{ margin: 0, color: "rgba(0, 0, 0, 0.88)" }}>
                        {journal.symbol || "待填写标的"}
                      </Title>
                      <Button
                        size="small"
                        type="text"
                        aria-label="编辑标的"
                        icon={<EditOutlined />}
                        onClick={() => {
                          setSymbolDraft(journal.symbol.trim() || recentSymbol);
                          setIsEditingSymbol(true);
                        }}
                      />
                    </>
                  )}
                  <Tag color={tagColorForVariant(journal.status === "open" ? "accent" : "neutral")}>
                    {journal.status === "open" ? "进行中" : "已完成"}
                  </Tag>
                </Flex>
                <Text style={{ color: "rgba(0, 0, 0, 0.45)" }}>
                  净仓 {formatNumber(netPosition(journal.events))}
                </Text>
              </Space>

            </Flex>

            {journal.summary ? (
              <div style={plainMutedBlockStyle}>
                <Paragraph style={{ margin: 0, color: "rgba(0, 0, 0, 0.45)", lineHeight: 1.8 }}>
                  {journal.summary}
                </Paragraph>
              </div>
            ) : null}

            {journal.tags.length > 0 ? (
              <Space orientation="vertical" size={8} style={{ width: "100%" }}>
                <Text strong style={{ color: "rgba(0, 0, 0, 0.88)" }}>
                  复盘标签
                </Text>
                <Space wrap size={[6, 6]}>
                  {journal.tags.map((tag) => (
                    <Tag key={tag}>{tag}</Tag>
                  ))}
                </Space>
              </Space>
            ) : null}

            {journal.images.length > 0 ? (
              <Space orientation="vertical" size={8} style={{ width: "100%" }}>
                <Flex justify="space-between" align="center" gap={12} wrap="wrap">
                  <Text strong style={{ color: "rgba(0, 0, 0, 0.88)" }}>
                    复盘图片
                  </Text>
                  <Tag color={tagColorForVariant("neutral")}>{journal.images.length} 张</Tag>
                </Flex>

                <Flex wrap="wrap" gap={8}>
                  {visibleImages.map((image) => {
                    const src = imageUrls[image.id];

                    return (
                      <Button
                        key={image.id}
                        type="link"
                        style={{ padding: 0, height: "auto", borderRadius: 10, overflow: "hidden" }}
                        onClick={() => {
                          void openImagePreview(image);
                        }}
                      >
                        <div
                          style={{
                            width: 84,
                            height: 64,
                            background: "rgba(15, 23, 42, 0.04)",
                            border: "1px solid rgba(15, 23, 42, 0.08)",
                            display: "flex",
                            alignItems: "center",
                            justifyContent: "center",
                          }}
                        >
                          {src ? (
                            <img
                              alt={image.fileName}
                              src={src}
                              style={{ width: "100%", height: "100%", objectFit: "cover" }}
                            />
                          ) : (
                            <Text type="secondary" style={{ fontSize: 12 }}>
                              加载中
                            </Text>
                          )}
                        </div>
                      </Button>
                    );
                  })}
                  {overflowImageCount > 0 ? <Tag>+{overflowImageCount}</Tag> : null}
                </Flex>
              </Space>
            ) : null}
          </>
        ) : null}
      </Space>
    </div>
  );
}
