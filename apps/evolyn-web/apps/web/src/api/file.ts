import { http } from '@evolyn.do/utils';

interface UploadSession {
  fileId: string;
  upload: { method: string; url: string; headers: Record<string, string> };
}

/** 上传通用私有文件并返回统一内容地址。 */
export async function uploadFile(file: File): Promise<string> {
  const session = await http.post<UploadSession>('/files/uploads', {
    filename: file.name,
    contentType: file.type,
    size: file.size,
  });
  const response = await fetch(session.upload.url, {
    method: session.upload.method,
    headers: session.upload.headers,
    body: file,
  });
  if (!response.ok) throw new Error('文件上传失败');
  await http.post(`/files/${session.fileId}/complete`);
  return `/api/v1/files/${session.fileId}/content`;
}

/** 兼容既有账号头像调用。 */
export function uploadAvatar(file: File): Promise<string> {
  return uploadFile(file);
}
