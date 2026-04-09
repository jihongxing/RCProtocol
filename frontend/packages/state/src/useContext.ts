import { ref } from 'vue'
import type { OrgContext, BrandContext } from './types'

const currentOrg = ref<OrgContext | null>(null)
const currentBrand = ref<BrandContext | null>(null)

export function useContext() {
  function setOrg(org: OrgContext) {
    currentOrg.value = org
  }

  function setBrand(brand: BrandContext) {
    currentBrand.value = brand
  }

  return { currentOrg, currentBrand, setOrg, setBrand }
}
