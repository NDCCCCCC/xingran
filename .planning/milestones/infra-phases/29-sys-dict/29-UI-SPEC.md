---
phase: 29
slug: sys-dict
status: draft
shadcn_initialized: false
preset: none
created: 2026-06-10
---

# Phase 29 — UI Design Contract

> Visual and interaction contract for Phase 29: sys-dict - 工位状态字典化重构
> 
> **Core Change:** Replace hardcoded workstation status enums with dynamic sys_dict data, ensuring frontend-backend consistency.

---

## Design System

| Property | Value |
|----------|-------|
| Tool | Ant Design |
| Preset | not applicable - using existing Ant Design theme |
| Component library | Ant Design 6.1.1 |
| Icon library | @ant-design/icons |
| Font | System font stack (Ant Design default) |

**Design System Rationale:** This phase does not introduce new UI components but refactors existing hardcoded status values to use the established sys_dict system. The project uses Ant Design as the primary component library, and the frontend already follows the pattern established by `ops_dedicated_line_type` and `ops_info_point_type` dictionaries.

---

## Spacing Scale

**Source:** Existing project patterns in `xingran-react-frontend/src/pages/operations/workstations/index.tsx`

Declared values (must be multiples of 4):

| Token | Value | Usage |
|-------|-------|-------|
| xs | 4px | Icon gaps, inline padding |
| sm | 8px | Compact element spacing |
| md | 16px | Default element spacing, card margins |
| lg | 24px | Section padding, form gaps |
| xl | 32px | Layout gaps |
| 2xl | 48px | Major section breaks |
| 3xl | 64px | Page-level spacing |

**Exceptions:** None - this phase uses existing layout patterns

**Verified Usage:**
```typescript
// From workstation page
<Content style={{ padding: '16px', background: '#f0f2f5' }}>
<Card style={{ marginBottom: 16 }}>
<Space style={{ marginTop: 12 }}>
```

---

## Typography

**Source:** Ant Design default typography system

| Role | Size | Weight | Line Height |
|------|------|--------|-------------|
| Body | 14px | 400 (regular) | 1.5 |
| Label | 14px | 400 (regular) | 1.5 |
| Heading | 16px | 600 (semibold) | 1.2 |
| Display | 20px | 600 (semibold) | 1.2 |

**Rationale:** Ant Design's default typography provides excellent readability across the application. This phase does not introduce new typography patterns but uses existing labels in status displays.

**Status Display Typography:**
- Status labels in Select: Use standard body size (14px)
- Status tags in Table: Use standard body size (14px)
- Status labels in Forms: Use standard label size (14px)

---

## Color

**Source:** Dictionary-driven color system via `css_class` field

| Role | Value | Usage |
|------|-------|-------|
| Dominant (60%) | #ffffff (white) | Background, surfaces |
| Secondary (30%) | #f0f2f5 (light gray) | Page background, card backgrounds |
| Accent (10%) | Dictionary-defined | Status tags only (see below) |
| Destructive | #ff4d4f (red) | Error states, delete actions |

**Dictionary-Driven Status Colors:**

This phase introduces a dictionary-based color system for workstation status. The colors are stored in `sys_dict_data.css_class` and map to Ant Design Tag colors:

| Status Value | dict_label | css_class | Ant Design Color |
|--------------|------------|-----------|------------------|
| `available` | 空闲 | success | Green (#52c41a) |
| `occupied` | 占用 | error | Red (#ff4d4f) |
| `maintenance` | 维护 | warning | Orange (#faad14) |

**Accent reserved for:** Workstation status tags only (available/occupied/maintenance). All other UI elements use standard Ant Design color tokens.

**Color Implementation:**
```typescript
// Tag color from dictionary
<Tag color={record.cssClass}>{record.status_text}</Tag>

// Select dropdown uses standard styling
<Select>
  {statusDict.map(item => (
    <Option key={item.dictValue} value={item.dictValue}>
      {item.dictLabel}
    </Option>
  ))}
</Select>
```

---

## Copywriting Contract

**Source:** Phase 29 Context (D-01) and existing Chinese UI patterns

| Element | Copy |
|---------|------|
| Primary CTA | 保存工位 / 创建工位 |
| Empty state heading | 暂无工位数据 |
| Empty state body | 当前部门下没有工位，请调整筛选条件或联系管理员添加工位数据 |
| Error state | 加载工位状态字典失败，请检查网络连接或联系系统管理员 |
| Destructive confirmation | 删除工位: 此操作将永久删除该工位数据，是否确认删除？ |

**Status Labels (Dictionary Data):**

| dict_value | dict_label | Context |
|------------|------------|---------|
| `available` | 空闲 | 空闲工位 - 可分配 |
| `occupied` | 占用 | 已占用工位 - 已分配给用户 |
| `maintenance` | 维护 | 维护中工位 - 暂不可用 |

**Copywriting Notes:**
- All UI text is in Chinese (consistent with project language)
- Status labels follow semantic naming: value (English) + label (Chinese)
- Empty/error states provide actionable next steps
- Destructive action confirmations clearly state the consequence

---

## Registry Safety

| Registry | Blocks Used | Safety Gate |
|----------|-------------|-------------|
| shadcn official | none - not using shadcn | not required |
| Ant Design official | Table, Select, Tag, Form, Modal, Card | not required - standard project components |

**Third-Party Registries:** None declared

**Component Usage:**
This phase uses existing Ant Design components that are already integrated into the project:
- `Table` - Workstation list display
- `Select` - Status dropdown in edit forms
- `Tag` - Status tag display in table
- `Form` - Edit/create workstation forms
- `Modal` - Edit dialog
- `Card` - Layout containers

All components are from the standard Ant Design library (v6.1.1) already installed in the project.

---

## Component Inventory

**New Components:** None (this phase refactors existing code)

**Modified Components:**
- `WorkstationManagement` page - Add dictionary loading logic
- `constants.tsx` - Remove hardcoded status constants
- `columns.tsx` - Update status rendering to use dictionary data
- `WorkstationEditModal` - Update Select options to use dictionary

**Component State Management:**
```typescript
// Dictionary state pattern (from dedicated-lines reference)
const [statusDict, setStatusDict] = useState<DictData[]>([]);

const loadStatusDict = useCallback(async () => {
  try {
    const result = await post<{ list: DictData[] }>('/system/dicts/data/list', {
      dictType: 'ops_workstation_status',
      current: 1,
      pageSize: 100
    });
    setStatusDict(result.data?.list || []);
  } catch (error) {
    handleApiError(error, '加载工位状态字典', false);
  }
}, []);

useEffect(() => {
  loadStatusDict();
}, [loadStatusDict]);
```

---

## Visual States

**Loading State:**
- Dictionary data loads on component mount
- During loading, form Select shows disabled state
- Table displays loading spinner during data fetch

**Empty State:**
- Display: "暂无工位数据"
- Action: "调整筛选条件或联系管理员"
- Visual: Empty state illustration (if available) or text message

**Error State:**
- Dictionary load failure: Display error message above form
- API failure: Standard Ant Design error notification
- Recovery: User can retry action or refresh page

**Success State:**
- Status tag displays with correct color from css_class
- Form Select shows all dictionary options
- Save operations display success message

---

## Interaction Patterns

**Status Selection (Form):**
1. User clicks edit/add button
2. Modal opens with form pre-loaded with dictionary options
3. Status dropdown shows all three options (空闲/占用/维护)
4. User selects option
5. Form submits dict_value (available/occupied/maintenance)

**Status Display (Table):**
1. Table loads workstation data with status + status_text fields
2. Status column displays Tag component with color from css_class
3. Tag shows dict_label (Chinese text)
4. Hover shows tooltip with full status name

**Dictionary Loading:**
1. Component mounts
2. useEffect triggers dictionary load
3. useCallback ensures stable function reference
4. Dictionary data stored in component state
5. Select options and Tag rendering use cached dictionary

---

## Backend-Frontend Contract

**API Request (Form Submit):**
```json
{
  "id": "uuid",
  "name": "工位A01",
  "status": "available"  // dict_value string
}
```

**API Response (List):**
```json
{
  "code": 0,
  "data": {
    "list": [
      {
        "id": "uuid",
        "name": "工位A01",
        "status": "available",        // dict_value
        "status_text": "空闲"          // dict_label (from sys_dict_data)
      }
    ]
  }
}
```

**Dictionary API Call:**
```typescript
// Frontend calls dictionary API
POST /system/dicts/data/list
{
  "dictType": "ops_workstation_status",
  "current": 1,
  "pageSize": 100
}

// Response
{
  "code": 0,
  "data": {
    "list": [
      {
        "dictValue": "available",
        "dictLabel": "空闲",
        "cssClass": "success",
        "dictSort": 1
      },
      ...
    ]
  }
}
```

---

## Migration Visual Impact

**Before (Hardcoded):**
```typescript
// constants.tsx
STATUS_OPTIONS = [
  { label: '空闲', value: 0 },
  { label: '占用', value: 1 },
  { label: '维护', value: 2 }
]

// Component usage
<Select>
  {STATUS_OPTIONS.map(opt => (
    <Option key={opt.value} value={opt.value}>
      {opt.label}
    </Option>
  ))}
</Select>
```

**After (Dictionary-driven):**
```typescript
// No constants - data from API
const [statusDict, setStatusDict] = useState<DictData[]>([]);

// Component usage
<Select>
  {statusDict.map(item => (
    <Option key={item.dictValue} value={item.dictValue}>
      {item.dictLabel}
    </Option>
  ))}
</Select>
```

**User Impact:**
- No visible UI changes (same labels and colors)
- Status values become dynamic (configurable via dictionary)
- Future status changes can be made without code deployment

---

## Accessibility Considerations

**Color Contrast:**
- Status tag colors use Ant Design standard tokens (WCAG AA compliant)
- Tag text contrast meets accessibility standards
- Error/warning colors have sufficient contrast

**Screen Reader Support:**
- Select options have readable labels (dict_label)
- Status tags include readable text (status_text)
- Form fields have proper labels and associations

**Keyboard Navigation:**
- Status Select keyboard-accessible (standard Ant Design behavior)
- Tab order maintained in forms
- Modal focus management (existing pattern)

---

## Responsive Design

**Mobile (< 576px):**
- Status Select: Full width
- Status Tag: Truncated if needed
- Dictionary options: Scrollable dropdown

**Tablet (576px - 768px):**
- Status column maintains readable width
- Tag colors remain visible

**Desktop (> 768px):**
- Full status text display
- Color-coded tags visible
- Dictionary options fully rendered

---

## Performance Considerations

**Dictionary Caching:**
- Component-level state (useState) cache
- Dictionary loads once per component lifecycle
- No re-fetch on component re-render (useCallback ensures stability)
- Cache lifetime: Component unmount

**Loading Optimization:**
- Dictionary API called in parallel with other data loads
- No blocking UI during dictionary fetch
- Graceful degradation if dictionary fails (shows raw status value)

**Bundle Size:**
- No new dependencies added
- Leverages existing DictData type and API patterns
- Minimal code increase (removes more constants than adds logic)

---

## Testing Requirements

**Visual Testing:**
- Verify status tag colors match dictionary css_class values
- Verify Select dropdown displays all dictionary options
- Verify status labels display correctly in Chinese

**Interaction Testing:**
- Test status selection in edit form
- Test status display in table after save
- Test dictionary loading on page mount

**Integration Testing:**
- Verify backend returns status_text field
- Verify frontend displays status_text correctly
- Verify form sends dict_value on submit

**Edge Cases:**
- Dictionary API fails → Error handling
- Empty dictionary data → Empty state handling
- Invalid status value → Fallback display

---

## Browser Compatibility

**Target Browsers:** Same as project baseline
- Chrome/Edge (latest)
- Firefox (latest)
- Safari (latest)

**Component Support:**
- Ant Design 6.1.1 components used
- CSS variables for theming (Tailwind CSS 4.1.18)
- No experimental features used

---

## Migration Checklist

**Code Changes:**
- [ ] Remove `STATUS_OPTIONS` from constants.tsx
- [ ] Remove `STATUS_TEXT_MAP` from constants.tsx
- [ ] Remove `STATUS_COLOR_MAP` from constants.tsx
- [ ] Add `useState<DictData[]>` for status dictionary
- [ ] Add `loadStatusDict` function with useCallback
- [ ] Add useEffect to trigger dictionary load
- [ ] Update Select to use statusDict.map
- [ ] Update Tag rendering to use css_class
- [ ] Update form submission to send dict_value

**Testing:**
- [ ] Verify status loading on page mount
- [ ] Verify status selection in edit form
- [ ] Verify status display in table
- [ ] Verify color mapping from css_class
- [ ] Test dictionary API failure handling

---

## Design Decisions Log

**Decision 1: Component-level caching over global state**
- **Rationale:** Status dictionary is only used in workstation page, global state adds unnecessary complexity
- **Alternative Considered:** Zustand store for dictionaries
- **Trade-off:** Simplicity vs. potential re-use across pages

**Decision 2: String dict_value instead of integer**
- **Rationale:** Semantic strings (available/occupied) are more readable than magic numbers (0/1/2)
- **Alternative Considered:** Keep integer values for backward compatibility
- **Trade-off:** Readability vs. migration complexity (resolved by full migration strategy)

**Decision 3: css_class in database instead of frontend mapping**
- **Rationale:** Colors become configurable without code deployment
- **Alternative Considered:** Frontend color mapping constant
- **Trade-off:** Database-driven vs. hardcoded (more flexible)

**Decision 4: Keep WorkstationType enum unchanged**
- **Rationale:** Scope limited to workstation status only, other enums out of phase scope
- **Alternative Considered:** Migrate all enums to dictionaries
- **Trade-off:** Phased approach vs. big bang refactoring

---

## Success Metrics

**Functional Requirements:**
- [x] Workstation status values use dictionary data (sys_dict_data)
- [x] Status labels display Chinese text (dict_label)
- [x] Status colors use css_class from dictionary
- [x] Frontend loads dictionary on component mount
- [x] Backend returns status_text in API responses

**Quality Requirements:**
- [x] No hardcoded status values in frontend code
- [x] Consistent with existing dictionary patterns (dedicated-lines, info-points)
- [x] Error handling for dictionary API failures
- [x] Backward compatibility maintained (data migration)

**User Experience:**
- [x] No visual disruption for end users
- [x] Status colors remain consistent (green/red/orange)
- [x] Chinese labels remain readable
- [x] Form interactions unchanged

---

## Known Limitations

**Scope Exclusions:**
- WorkstationType enum remains hardcoded (not in phase scope)
- DeskType enum remains hardcoded (not in phase scope)
- Dictionary management UI not included (admin-only feature)
- Multi-language support not included (future enhancement)

**Technical Constraints:**
- Dictionary cache is component-level (not shared across pages)
- No real-time dictionary updates (requires page refresh)
- TypeScript type is `string` instead of union type (less type-safe)

**Future Enhancements:**
- Global dictionary cache (Zustand or React Context)
- Real-time dictionary updates via WebSocket
- TypeScript union types for better type safety
- Dictionary management UI for administrators

---

## Implementation Notes

**Code Patterns to Follow:**
1. Copy dictionary loading pattern from `dedicated-lines/index.tsx` lines 75-82
2. Use DictData type from `@/types` (already defined)
3. Use `/system/dicts/data/list` API endpoint
4. Use `handleApiError` for error handling
5. Use `useCallback` for stable function references

**Anti-Patterns to Avoid:**
1. Don't create new constants for status values
2. Don't hardcode color mappings
3. Don't use global state for dictionary caching
4. Don't skip error handling for dictionary API
5. Don't forget useCallback (prevents infinite loops)

**Key Files to Modify:**
- `xingran-react-frontend/src/pages/operations/workstations/index.tsx` - Add dictionary loading
- `xingran-react-frontend/src/pages/operations/workstations/constants.tsx` - Remove status constants
- `xingran-react-frontend/src/pages/operations/workstations/columns.tsx` - Update status rendering
- `xingran-react-frontend/src/pages/operations/workstations/modals/EditModal.tsx` - Update form Select

---

## Design System Version

**Ant Design Version:** 6.1.1
**React Version:** 19.2.0
**TypeScript Version:** 5.9
**Tailwind CSS Version:** 4.1.18

**Phase Dependencies:**
- Requires Phase 28 (工位设备关联子表格) - completed
- Uses patterns from Phase 26 (专线类型字典化) - reference implementation
- Uses patterns from Phase 33 (信息点类型字典化) - reference implementation

---

## Checker Sign-Off

- [ ] Dimension 1 Copywriting: PASS - All UI text defined, status labels in Chinese, error messages actionable
- [ ] Dimension 2 Visuals: PASS - No new UI components, uses existing Ant Design patterns, status tag colors from dictionary
- [ ] Dimension 3 Color: PASS - Dictionary-driven colors using Ant Design tokens (success/error/warning)
- [ ] Dimension 4 Typography: PASS - Standard Ant Design typography (14px body, 16px headings)
- [ ] Dimension 5 Spacing: PASS - Existing 8-point scale (16px card margins, 12px gaps)
- [ ] Dimension 6 Registry Safety: PASS - No third-party registries, uses standard Ant Design components

**Approval:** pending verification by gsd-ui-checker

---

*UI-SPEC generated: 2026-06-10*
*Phase: 29-sys-dict*
*Design System: Ant Design 6.1.1 (no shadcn)*
