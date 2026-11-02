// 湖北省主要城市坐标
export const HUBEI_CITIES = {
  wuhan: { name: "武汉市", center: [114.305393, 30.593099] },
  huangshi: { name: "黄石市", center: [115.038917, 30.201694] },
  shiyan: { name: "十堰市", center: [110.787516, 32.629247] },
  yichang: { name: "宜昌市", center: [111.286768, 30.691845] },
  xiangyang: { name: "襄阳市", center: [112.144146, 32.042426] },
  ezhou: { name: "鄂州市", center: [114.895305, 30.393495] },
  jingmen: { name: "荆门市", center: [112.199235, 31.035419] },
  xiaogan: { name: "孝感市", center: [113.916969, 30.924277] },
  jingzhou: { name: "荆州市", center: [112.241347, 30.332591] },
  huanggang: { name: "黄冈市", center: [114.872856, 30.453648] },
  xianning: { name: "咸宁市", center: [114.322389, 29.841294] },
  suizhou: { name: "随州市", center: [113.382652, 31.690332] },
  enshi: { name: "恩施州", center: [109.487966, 30.283072] },
  xiantao: { name: "仙桃市", center: [113.454238, 30.364035] },
  qianjiang: { name: "潜江市", center: [112.899379, 30.421735] },
  tianmen: { name: "天门市", center: [113.166138, 30.693062] },
  shennongjia: { name: "神农架", center: [110.675699, 31.744749] },
} as const;

// 城市类型
export type CityCode = keyof typeof HUBEI_CITIES;

// 获取城市信息
export function getCityInfo(code: string) {
  return HUBEI_CITIES[code as CityCode];
}

// 获取所有城市列表
export function getAllCities() {
  return Object.entries(HUBEI_CITIES).map(([code, info]) => ({
    code,
    ...info,
  }));
}
