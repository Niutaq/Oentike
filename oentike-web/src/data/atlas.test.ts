import { describe, it, expect } from 'vitest';
import { cardDisplayName, AtlasCard } from './atlas';

describe('cardDisplayName', () => {
    it('returns the English name when lang is en', () => {
        const mockCard = { names: { pl: 'Borowik', en: 'Penny Bun' } } as AtlasCard;
        expect(cardDisplayName(mockCard, 'en')).toBe('Penny Bun');
        expect(cardDisplayName(mockCard, 'en-US')).toBe('Penny Bun');
    });

    it('returns the Polish name when lang is not en', () => {
        const mockCard = { names: { pl: 'Borowik', en: 'Penny Bun' } } as AtlasCard;
        expect(cardDisplayName(mockCard, 'pl')).toBe('Borowik');
        expect(cardDisplayName(mockCard, 'de')).toBe('Borowik'); // fallback to PL
    });
});
