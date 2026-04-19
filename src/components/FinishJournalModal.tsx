import { Form, Input, Modal } from "antd";
import { useEffect } from "react";

export function FinishJournalModal({
  open,
  initialSummary,
  onClose,
  onConfirm,
}: {
  open: boolean;
  initialSummary: string;
  onClose: () => void;
  onConfirm: (summary: string) => void;
}) {
  const [form] = Form.useForm<{ summary: string }>();

  useEffect(() => {
    if (open) {
      form.setFieldsValue({ summary: initialSummary });
    }
  }, [form, initialSummary, open]);

  return (
    <Modal
      title="结束并写下总结"
      open={open}
      destroyOnHidden
      okText="确认结束"
      cancelText="取消"
      onCancel={onClose}
      onOk={() =>
        form.validateFields().then((values) => {
          onConfirm(values.summary ?? "");
        })
      }
    >
      <Form form={form} layout="vertical">
        <Form.Item name="summary" label="复盘摘要">
          <Input.TextArea rows={5} placeholder="可选：这笔交易哪里做得好，哪里需要改进？" />
        </Form.Item>
      </Form>
    </Modal>
  );
}
