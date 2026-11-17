import {describe,it,expect} from 'vitest';
describe('Dhaba Orders UI',()=>{it('formats the PKR currency',()=>{expect(new Intl.NumberFormat('en-PK',{style:'currency',currency:'PKR',maximumFractionDigits:0}).format(420)).toContain('420')})});
