// Phase 90 矩阵 fixture：22 形态注入 antd cssVar scope
// 源自 89-visual-evidence/matrix-fixture.js（21 形态），仅增补 f22_input_number_status_error，
// 并把采样属性由 5 个扩为 6 个（追加 fontSize）。f01-f21 的 forms/carrierSel/innerSel 逐字沿用 89。
(function () {
  var scope = document.querySelector('.ant-app, [class*="css-var-"]');
  if (!scope) return JSON.stringify({ error: 'no cssVar scope found' });
  var prev = document.getElementById('__p88matrix');
  if (prev) prev.remove();
  var c = document.createElement('div');
  c.id = '__p88matrix';
  c.style.cssText = 'position:fixed;left:-9999px;top:0;width:1200px;';
  scope.appendChild(c);
  var S = '<span class="ant-select-selection-search"><input class="ant-select-selection-search-input"></span>';
  var forms = {
    f01_input_plain: '<input class="ant-input" value="">',
    f02_input_prefix: '<span class="ant-input-affix-wrapper"><span class="ant-input-prefix">P</span><input class="ant-input" value=""></span>',
    f03_input_suffix: '<span class="ant-input-affix-wrapper"><input class="ant-input" value=""><span class="ant-input-suffix">S</span></span>',
    f04_input_allowclear: '<span class="ant-input-affix-wrapper ant-input-affix-wrapper"><input class="ant-input" value=""><span class="ant-input-suffix"><span class="ant-input-clear-icon"></span></span></span>',
    f05_input_addonbefore: '<span class="ant-input-group-wrapper"><span class="ant-input-wrapper"><span class="ant-input-group-addon">http://</span><input class="ant-input" value=""></span></span>',
    f06_input_addonafter: '<span class="ant-input-group-wrapper"><span class="ant-input-wrapper"><input class="ant-input" value=""><span class="ant-input-group-addon">.com</span></span></span>',
    f07_input_password: '<span class="ant-input-affix-wrapper ant-input-password"><input type="password" class="ant-input" value=""><span class="ant-input-suffix"><span class="ant-input-password-icon"></span></span></span>',
    f08_input_search: '<span class="ant-input-group-wrapper"><span class="ant-input-wrapper"><span class="ant-input-affix-wrapper"><input class="ant-input" value=""></span><span class="ant-input-group-addon"><button class="ant-btn ant-btn-primary ant-input-search-button">Q</button></span></span></span>',
    f09_textarea: '<span class="ant-input-affix-wrapper"><textarea class="ant-input" rows="1"></textarea></span>',
    f10_textarea_showcount: '<span class="ant-input-affix-wrapper"><span class="ant-input-affix-wrapper"><textarea class="ant-input" rows="1"></textarea><span class="ant-input-data-count">0 / 100</span></span></span>',
    f11_input_number: '<div class="ant-input-number ant-input-number-outlined"><div class="ant-input-number-input-wrap"><input class="ant-input-number-input" value="1"></div></div>',
    f12_select_single: '<div class="ant-select ant-select-outlined ant-select-in-form-item ant-select-single ant-select-show-arrow"><div class="ant-select-content"><div class="ant-select-placeholder">请选择</div><input class="ant-select-input" readonly></div><span class="ant-select-arrow"></span></div>',
    f13_select_multiple: '<div class="ant-select ant-select-outlined ant-select-in-form-item ant-select-multiple ant-select-show-arrow"><div class="ant-select-content"><div class="ant-select-placeholder">请选择</div><input class="ant-select-input" readonly></div><span class="ant-select-arrow"></span></div>',
    f14_datepicker: '<div class="ant-picker ant-picker-outlined"><div class="ant-picker-input"><input class="ant-input" value=""></div><span class="ant-picker-suffix">C</span></div>',
    f15_range_picker: '<div class="ant-picker ant-picker-range ant-picker-outlined"><div class="ant-picker-input"><input class="ant-input" value=""></div><div class="ant-picker-range-separator">-</div><div class="ant-picker-input"><input class="ant-input" value=""></div><span class="ant-picker-suffix">C</span></div>',
    f16_cascader: '<div class="ant-select ant-cascader ant-select-single ant-cascader-single ant-select-outlined ant-select-in-form-item"><div class="ant-select-content"><div class="ant-select-placeholder">请选择</div><input class="ant-select-input" readonly></div><span class="ant-select-arrow"></span></div>',
    f17_treeselect: '<div class="ant-select ant-tree-select ant-select-single ant-select-outlined ant-select-in-form-item"><div class="ant-select-content"><div class="ant-select-placeholder">请选择</div><input class="ant-select-input" readonly></div><span class="ant-select-arrow"></span></div>',
    f18_autocomplete: '<div class="ant-select ant-select-auto-complete ant-select-single ant-select-outlined"><div class="ant-select-content"><input class="ant-select-input" value=""></div></div>',
    f19_mentions: '<div class="ant-mentions"><textarea class="ant-mentions" rows="1"></textarea></div>',
    f20_appinput: '<span class="ant-input-affix-wrapper ant-input-affix-wrapper"><input class="ant-input" value=""><span class="ant-input-suffix"><span class="ant-input-clear-icon"></span></span></span>',
    f21_apptextarea: '<span class="ant-input-affix-wrapper"><span class="ant-input-affix-wrapper"><textarea class="ant-input" rows="1"></textarea><span class="ant-input-suffix"><span class="ant-input-clear-icon"></span></span></span></span>'
  };
  // Phase 90 新增形态 f22（VISFIX-05 的 b 类落点）：
  // antd 6.6.4 真实 status 类名 `ant-input-number-status-error`
  // （getStatusClassNames → `${prefixCls}-status-error`，落在 .ant-input-number 根；边框也画在根）。
  // 以「字面量之后追加赋值」写入 → 保证 f01-f21 的定义逐字未改（基线不漂移，可机械核对）。
  forms.f22_input_number_status_error = '<div class="ant-input-number ant-input-number-outlined ant-input-number-status-error"><div class="ant-input-number-input-wrap"><input class="ant-input-number-input" value="1"></div></div>';
  var carrierSel = {
    f01_input_plain: 'self', f02_input_prefix: '.ant-input-affix-wrapper', f03_input_suffix: '.ant-input-affix-wrapper',
    f04_input_allowclear: '.ant-input-affix-wrapper', f05_input_addonbefore: '.ant-input', f06_input_addonafter: '.ant-input',
    f07_input_password: '.ant-input-affix-wrapper', f08_input_search: '.ant-input-affix-wrapper', f09_textarea: '.ant-input-affix-wrapper',
    f10_textarea_showcount: '.ant-input-affix-wrapper .ant-input-affix-wrapper', f11_input_number: '.ant-input-number',
    f12_select_single: 'self', f13_select_multiple: 'self', f14_datepicker: '.ant-picker',
    f15_range_picker: '.ant-picker', f16_cascader: 'self', f17_treeselect: 'self',
    f18_autocomplete: 'self', f19_mentions: 'textarea.ant-mentions', f20_appinput: '.ant-input-affix-wrapper',
    f21_apptextarea: '.ant-input-affix-wrapper .ant-input-affix-wrapper'
  };
  carrierSel.f22_input_number_status_error = '.ant-input-number';
  var innerSel = {
    f02_input_prefix: 'input.ant-input', f03_input_suffix: 'input.ant-input', f04_input_allowclear: 'input.ant-input',
    f05_input_addonbefore: 'input.ant-input', f06_input_addonafter: 'input.ant-input', f07_input_password: 'input.ant-input',
    f08_input_search: 'input.ant-input', f09_textarea: 'textarea.ant-input', f10_textarea_showcount: 'textarea.ant-input',
    f11_input_number: '.ant-input-number-input', f14_datepicker: 'input.ant-input', f15_range_picker: 'input.ant-input',
    f18_autocomplete: 'input.ant-input', f19_mentions: 'textarea.ant-mentions', f20_appinput: 'input.ant-input',
    f21_apptextarea: 'textarea.ant-input'
  };
  innerSel.f22_input_number_status_error = '.ant-input-number-input';
  var html = '';
  for (var k in forms) html += '<div data-f="' + k + '">' + forms[k] + '</div>';
  c.innerHTML = html;
  function pick(el, sel, isInner) {
    var t = sel === 'self' || !sel ? el : (el.matches(sel) ? el : el.querySelector(sel));
    if (!t) return null;
    var cs = getComputedStyle(t);
    return {
      bg: cs.backgroundColor, border: cs.borderTopColor, radius: cs.borderTopLeftRadius,
      shadow: cs.boxShadow, btw: cs.borderTopWidth, fontSize: cs.fontSize, cls: t.className
    };
  }
  var out = {};
  Array.prototype.forEach.call(c.children, function (wrap) {
    var id = wrap.dataset.f;
    out[id] = {
      carrier: pick(wrap.querySelector(':scope > *'), carrierSel[id], false),
      inner: innerSel[id] ? pick(wrap, innerSel[id], true) : null
    };
  });
  return JSON.stringify({ mode: document.documentElement.getAttribute('data-color-mode'), mountedIn: scope.className.slice(0, 60), forms: out });
})();
