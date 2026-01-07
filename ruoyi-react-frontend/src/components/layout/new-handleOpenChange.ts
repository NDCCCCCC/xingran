  // 处理菜单展开/收起 - 改进版手风琴效果（支持二级菜单手风琴）
  const handleOpenChange: MenuProps['onOpenChange'] = (keys) => {
    console.log('菜单展开/收起事件:', keys, '当前 openKeys:', openKeys, '预期:', expectedOpenKeysRef.current);

    // 检查是否是导航触发的预期变化
    if (expectedOpenKeysRef.current) {
      const expected = expectedOpenKeysRef.current;
      // 比较数组内容（忽略顺序）
      const isExpectedChange =
        keys.length === expected.length &&
        keys.every(k => expected.includes(k));

      if (isExpectedChange) {
        console.log('这是导航触发的预期变化，直接应用，跳过手风琴逻辑');
        setOpenKeys(keys);
        return;
      }
    }

    // 找出新展开的菜单（keys 中有但 openKeys 中没有的）
    const latestOpenKey = keys.find((key) => !openKeys.includes(key));

    if (latestOpenKey) {
      console.log('展开菜单:', latestOpenKey);

      const menuLevel = getMenuLevel(latestOpenKey, menuPathMap);
      console.log('菜单层级:', menuLevel);

      if (menuLevel === 1 || latestOpenKey.startsWith('top_')) {
        // ========================================
        // 一级菜单：应用手风琴效果
        // ========================================
        console.log('是一级菜单，应用手风琴效果');
        // 1. 保留新展开的一级菜单
        // 2. 保留所有非一级菜单（二级和三级菜单）
        // 3. 移除其他所有一级菜单
        const nonTopLevelKeys = keys.filter(key => !isTopLevelMenu(key, menuPathMap));
        const result = [latestOpenKey, ...nonTopLevelKeys];

        console.log('一级菜单手风琴结果:', result);
        setOpenKeys(result);

      } else if (menuLevel === 2) {
        // ========================================
        // 二级菜单：应用手风琴效果（仅在同一一级菜单下）
        // ========================================
        console.log('是二级菜单，应用手风琴效果（仅在同一一级菜单下）');

        // 1. 找出当前二级菜单所属的一级菜单
        const currentTopLevel = menuPathMap.get(latestOpenKey)?.topLevel;

        if (!currentTopLevel) {
          console.warn('无法确定二级菜单的所属一级菜单，直接应用');
          setOpenKeys([...keys]);
          return;
        }

        console.log('当前二级菜单所属一级菜单:', currentTopLevel);

        // 2. 保留所有一级菜单
        const topLevelKeys = keys.filter(key => isTopLevelMenu(key, menuPathMap));

        // 3. 找出所有与新展开的二级菜单属于同一一级菜单的其他已展开二级菜单
        const sameTopLevelSecondKeys = openKeys.filter(key => {
          if (!isSecondLevelMenu(key, menuPathMap)) return false;
          return isSameTopLevelMenu(key, latestOpenKey, menuPathMap);
        });

        console.log('同属一个一级菜单的其他二级菜单:', sameTopLevelSecondKeys);

        // 4. 需要收起的这些同级二级菜单
        const keysToClose = sameTopLevelSecondKeys;

        // 5. 构建新的openKeys：
        //    - 保留所有一级菜单
        //    - 保留不属于同一一级菜单的其他二级菜单
        //    - 保留新展开的二级菜单
        //    - 保留所有三级菜单
        //    - 移除需要收起的同级二级菜单
        const result = [
          ...topLevelKeys,
          ...keys.filter(key =>
            !keysToClose.includes(key) &&
            (isSecondLevelMenu(key, menuPathMap) || key === latestOpenKey)
          )
        ];

        console.log('二级菜单手风琴结果:', result);
        setOpenKeys(result);

      } else if (menuLevel === 3) {
        // ========================================
        // 三级菜单：不应用手风琴效果
        // ========================================
        console.log('是三级菜单，不应用手风琴效果，直接应用');
        setOpenKeys([...keys]);

      } else {
        // ========================================
        // 未知层级：直接应用（降级处理）
        // ========================================
        console.warn('未知菜单层级，直接应用:', latestOpenKey);
        setOpenKeys([...keys]);
      }
    } else {
      // ========================================
      // 收起操作：直接应用
      // ========================================
      console.log('收起操作，直接应用:', keys);
      setOpenKeys([...keys]);
    }
  };
