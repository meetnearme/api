/* eslint-disable */
/**
 * This file was automatically generated by json-schema-to-typescript.
 * DO NOT MODIFY IT BY HAND. Instead, modify the source JSONSchema file,
 * and run json-schema-to-typescript to regenerate this file.
 */

export type Event = Event1 & Event2;
export type Event2 = {
  [k: string]: unknown;
};

export interface Event1 {
  id: string;
  ownerId?: string;
  name: string;
  description?: string;
  startTime: string;
  endTime?: string;
  eventCapacity?: number;
  eventCost?: number;
  recurrenceRule?: string;
  recurrenceId?: string;
  url: string;
  urlDomain?: string;
  urlPath?: string;
  urlQueryParams?: {
    [k: string]: unknown;
  };
  locationLatitude: number;
  locationLongitude: number;
  locationAddress?: string;
  locationCountry?: string;
  locationZipCode?: string;
  tags: string[];
  categories: string[];
  createdAt: number;
  updatedAt: number;
  [k: string]: unknown;
}
