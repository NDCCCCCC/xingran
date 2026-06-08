const assetCrudApi = createCrudApi<Asset>({ basePath: '/ops/asset' });

export const assetApi = {
  ...assetCrudApi,

  // Excel导入导出通过 SetupExcelRouter 注册在 /ops/asset 路由组下：
  // - GET  /ops/asset/template  (下载模板)
  // - POST /ops/asset/import    (上传导入)
  // - POST /ops/asset/export    (导出数据)

  excel: {
    template: async () => {
      return await post('/ops/asset/template', {});
    },

    import: async (file: File) => {
      const formData = new FormData();
      formData.append('file', file);
      return await postFormData('/ops/asset/import', formData);
    },

    export: async (params: AssetListParamsExtended) => {
      const token = await getAccessToken();
      const response = await fetch('/api/v1/ops/asset/export', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(params),
      });

      if (!response.ok) {
        throw new Error('导出失败');
      }

      const blob = await response.blob();
      const filename = `资产列表_${Date.now()}.xlsx`;
      const blobUrl = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = blobUrl;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(blobUrl);
      document.body.removeChild(a);

      return { code: 0, message: '导出成功', data: null };
    },
  },
};
