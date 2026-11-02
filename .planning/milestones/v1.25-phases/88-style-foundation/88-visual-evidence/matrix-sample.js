// Phase 88-03 采样脚本：对已挂载 fixture 逐格读 4 元组
(function () {
  var c = document.getElementById('__p88matrix');
  if (!c) return JSON.stringify({ error: 'fixture not mounted' });
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
  var innerSel = {
    f02_input_prefix: 'input.ant-input', f03_input_suffix: 'input.ant-input', f04_input_allowclear: 'input.ant-input',
    f05_input_addonbefore: 'input.ant-input', f06_input_addonafter: 'input.ant-input', f07_input_password: 'input.ant-input',
    f08_input_search: 'input.ant-input', f09_textarea: 'textarea.ant-input', f10_textarea_showcount: 'textarea.ant-input',
    f11_input_number: '.ant-input-number-input', f14_datepicker: 'input.ant-input', f15_range_picker: 'input.ant-input',
    f18_autocomplete: 'input.ant-input', f19_mentions: 'textarea.ant-mentions', f20_appinput: 'input.ant-input',
    f21_apptextarea: 'textarea.ant-input'
  };
  function pick(root, sel) {
    var t = sel === 'self' || !sel ? root : (root.matches(sel) ? root : root.querySelector(sel));
    if (!t) return null;
    var cs = getComputedStyle(t);
    return { bg: cs.backgroundColor, border: cs.borderTopColor, radius: cs.borderTopLeftRadius, shadow: cs.boxShadow, btw: cs.borderTopWidth };
  }
  var out = {};
  Array.prototype.forEach.call(c.children, function (wrap) {
    var id = wrap.dataset.f;
    out[id] = {
      carrier: pick(wrap.querySelector(':scope > *'), carrierSel[id]),
      inner: innerSel[id] ? pick(wrap, innerSel[id]) : null
    };
  });
  return JSON.stringify({ mode: document.documentElement.getAttribute('data-color-mode'), forms: out });
})();
