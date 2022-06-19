SET CLIENT_ENCODING TO 'UTF8';

CREATE EXTENSION IF NOT EXISTS lo;

CREATE TABLE public.updates 
(
    id bigserial NOT NULL,
    record_time timestamp without time zone NOT NULL DEFAULT Now(),
    channel text NOT NULL,    
    major integer NOT NULL,
    minor integer NOT NULL DEFAULT 0,
    patch integer NOT NULL DEFAULT 0,
    revision integer NOT NULL DEFAULT 0,
    build_time timestamp without time zone NOT NULL DEFAULT Now(),
    info text,
    enabled boolean NOT NULL DEFAULT true,    

    PRIMARY KEY (id)
);

COMMENT ON TABLE public.updates IS 'информация об обновлениях';
COMMENT ON COLUMN public.updates.record_time IS 'время добавления';
COMMENT ON COLUMN public.updates.channel IS 'канал обновлений (вид программы + окружение, например HRFILE_PROD)';
COMMENT ON COLUMN public.updates.major IS 'номер версии major';
COMMENT ON COLUMN public.updates.minor IS 'номер версии minor';
COMMENT ON COLUMN public.updates.patch IS 'номер версии patch';
COMMENT ON COLUMN public.updates.revision IS 'номер версии revision';
COMMENT ON COLUMN public.updates.build_time IS 'http url';
COMMENT ON COLUMN public.updates.info IS 'информация о версии';
COMMENT ON COLUMN public.updates.enabled IS 'включение и отключение обновления на эту версию';

CREATE INDEX idx_updates_version ON public.updates (channel, major, minor, patch, revision);
CREATE INDEX idx_updates_enabled ON public.updates (enabled);
ALTER TABLE IF EXISTS public.updates ADD CONSTRAINT uk_updates UNIQUE (channel, major, minor, patch, revision);

CREATE TABLE public.files
(    
    id_update bigint NOT NULL,
    file_name text NOT NULL,
    checksum text NOT NULL, 
    data_oid oid NOT NULL, 

    PRIMARY KEY (id_update, file_name),
    CONSTRAINT fk_files_update FOREIGN KEY (id_update) REFERENCES public.updates (id) MATCH SIMPLE ON UPDATE NO ACTION ON DELETE CASCADE
);

CREATE INDEX idx_files_checksum ON public.files (file_name, checksum);

-- очистка large object при изменениях
CREATE TRIGGER t_files_clear_data BEFORE UPDATE OR DELETE ON public.files FOR EACH ROW EXECUTE FUNCTION lo_manage(data_oid);

COMMENT ON TABLE public.files IS 'информация о файлах';
COMMENT ON COLUMN public.files.file_name IS 'относительный путь к файлу';
COMMENT ON COLUMN public.files.checksum IS 'контрольная сумма';
COMMENT ON COLUMN public.files.data_oid IS 'ссылка на large object c данными';

CREATE TABLE public.cache
(     
    id bigserial NOT NULL,
    id_update_from bigint,
    id_update_to bigint NOT NULL,

    diff_oid oid,
    diff_info jsonb NOT NULL,

    PRIMARY KEY (id),
    CONSTRAINT fk_cache_update_from FOREIGN KEY (id_update_from) REFERENCES public.updates (id) MATCH SIMPLE ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT fk_cache_update_to FOREIGN KEY (id_update_to) REFERENCES public.updates (id) MATCH SIMPLE ON UPDATE NO ACTION ON DELETE CASCADE
);
-- обычный UNIQUE CONSTRAINT не учитывает NULL
CREATE UNIQUE INDEX uk_cache ON public.cache (COALESCE(id_update_from,-1), id_update_to);

-- очистка large object при изменениях
CREATE TRIGGER t_cache_clear_data BEFORE UPDATE OR DELETE ON public.cache FOR EACH ROW EXECUTE FUNCTION lo_manage(diff_oid);

COMMENT ON TABLE public.cache IS 'подготовленные diff';
COMMENT ON COLUMN public.cache.id_update_from IS 'с какого обновления';
COMMENT ON COLUMN public.cache.id_update_to IS 'на какое обновление';
COMMENT ON COLUMN public.cache.diff_oid IS 'ссылка на large object c zip архивом diff';
COMMENT ON COLUMN public.cache.diff_info IS 'json с информацией об обновлении';