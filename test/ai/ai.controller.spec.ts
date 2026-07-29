import { AiController } from '../../src/ai/ai.controller';
import { AiService } from '../../src/ai/ai.service';

describe('AiController', () => {
  let controller: AiController;
  let service: jest.Mocked<Pick<AiService, 'getAiLogs'>>;

  beforeEach(() => {
    service = { getAiLogs: jest.fn() };
    controller = new AiController(service as unknown as AiService);
  });

  it('gets the requested AI log page', async () => {
    service.getAiLogs.mockResolvedValue({
      items: [],
      total: 0,
      page: 2,
      limit: 50,
    });

    await controller.getAiLogs({ page: 2, limit: 50 });

    expect(service.getAiLogs).toHaveBeenCalledWith(2, 50);
  });
});
